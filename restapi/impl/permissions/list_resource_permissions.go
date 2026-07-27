package permissions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"

	"github.com/go-openapi/runtime/middleware"
)

func listResourcePermissionsOk(perms []*models.Permission) middleware.Responder {
	return permissions.NewListResourcePermissionsOK().WithPayload(
		&models.PermissionList{Permissions: perms},
	)
}

func listResourcePermissionsInternalServerError(reason string) middleware.Responder {
	return permissions.NewListResourcePermissionsInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// expandGroupPermissions replaces any permissions granted to a group in the result set with permissions for the
// individual group members. The list is deduplicated so that users who appear in the list multiple times after
// expansion will only appear once with the highest level of permission that they have. Note: when `expand_groups`
// is set to `true`, this handler may write to the database in some rare cases when a subject happens to be in
// a group that is being expanded and does not have a record in the permissions database yet. This is done so
// that the endpoint can return an ID for every subject in the response body.
func expandGroupPermissions(
	ctx context.Context,
	tx *sql.Tx,
	grouperClient grouper.Grouper,
	perms []*models.Permission,
	userSuffix string,
) ([]*models.Permission, error) {

	// Get the list of group IDs from the list of permissions.
	groupIDs := make([]models.ExternalSubjectID, 0, len(perms))
	for _, perm := range perms {
		if grouperClient.IsGroupSource(*perm.Subject.SubjectSourceID) {
			groupIDs = append(groupIDs, *perm.Subject.SubjectID)
		}
	}

	// There's nothing to do if there are no group permissions in the list.
	if len(groupIDs) == 0 {
		return perms, nil
	}

	// Get the list of members for each of the groups.
	membersOf := make(map[models.ExternalSubjectID][]*models.SubjectOut, len(groupIDs))
	for _, groupID := range groupIDs {
		members, err := grouperClient.ListUsersInGroup(ctx, groupID)
		if err != nil {
			logger.Log.Errorf("listing members of group %s: %v", groupID, err)
			return nil, err
		}
		membersOf[groupID] = members
	}

	// Create some maps for deduplication and caching.
	permFor := make(map[models.ExternalSubjectID]*models.Permission)
	subjectFor := make(map[models.ExternalSubjectID]*models.SubjectOut)

	// Load a map from permission level to precedence for reference.
	precedenceLevelFor, err := permsdb.LoadPermissionLevelPrecedence(ctx, tx)
	if err != nil {
		logger.Log.Errorf("loading permission level precedence map: %v", err)
		return nil, err
	}

	// Create a local function to get the permission level precedence for a permission object for clarity.
	getPrecedenceOf := func(perm *models.Permission) (int, error) {
		level := *perm.PermissionLevel
		precedence, ok := precedenceLevelFor[level]
		if !ok {
			return 0, fmt.Errorf("unrecognized permission level %s found in permission", level)
		}
		return precedence, nil
	}

	// Create a local function to get a subject for clarity.
	getSubject := func(in *models.SubjectOut) (*models.SubjectOut, error) {
		out, ok := subjectFor[*in.SubjectID]
		if ok {
			return out, nil
		}
		subject, err := permsdb.GetOrAddSubject(ctx, tx, *in.SubjectID, models.SubjectTypeUser, userSuffix)
		if err != nil {
			return nil, err
		}
		subject.SubjectSourceID = in.SubjectSourceID
		subjectFor[*in.SubjectID] = subject
		return subject, nil
	}

	// Create a local function to set the permission for a subject for clarity.
	setPerm := func(candidate *models.Permission) error {
		candidatePrecedence, err := getPrecedenceOf(candidate)
		if err != nil {
			return err
		}
		current := permFor[*candidate.Subject.SubjectID]
		if current == nil {
			permFor[*candidate.Subject.SubjectID] = candidate
			return nil
		}
		currentPrecedence, err := getPrecedenceOf(current)
		if err != nil {
			return err
		}
		if candidatePrecedence < currentPrecedence {
			permFor[*candidate.Subject.SubjectID] = candidate
		}
		return nil
	}

	// Populate the maps.
	for _, perm := range perms {
		members, isGroupPerm := membersOf[*perm.Subject.SubjectID]
		if isGroupPerm {
			for _, member := range members {
				subject, err := getSubject(member)
				if err != nil {
					logger.Log.Errorf("finding subject for subject ID %v: %v", member, err)
					return nil, err
				}
				candidate := &models.Permission{
					ID:              perm.ID,
					PermissionLevel: perm.PermissionLevel,
					Resource:        perm.Resource,
					Subject:         subject,
				}
				if err := setPerm(candidate); err != nil {
					logger.Log.Errorf("setting permission for subject ID %v: %v", member, err)
					return nil, err
				}
			}
		} else {
			if err := setPerm(perm); err != nil {
				logger.Log.Errorf("setting permission already in result set (%v): %v", perm, err)
				return nil, err
			}
		}
	}

	// Build the resulting permissions from the maps.
	expandedPerms := make([]*models.Permission, 0, len(permFor))
	for _, perm := range permFor {
		expandedPerms = append(expandedPerms, perm)
	}
	return expandedPerms, nil
}

// BuildListResourcePermissionsHandler builds the request handler for the list resource permissions endpoint.
func BuildListResourcePermissionsHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema, userSuffix string,
) func(permissions.ListResourcePermissionsParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.ListResourcePermissionsParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		expandGroups := *params.ExpandGroups
		resourceTypeName := params.ResourceType
		resourceName := params.ResourceName

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}
		defer permsdb.RollbackTx(tx)

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// List the permissions for the resource.
		perms, err := permsdb.ListResourcePermissions(ctx, tx, resourceTypeName, resourceName)
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// Expand groups in the response body if we're supposed to.
		if expandGroups {
			perms, err = expandGroupPermissions(ctx, tx, grouperClient, perms, userSuffix)
			if err != nil {
				logger.Log.Errorf("expanding group permissions: %v", err)
				return listResourcePermissionsInternalServerError(err.Error())
			}
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			logger.Log.Errorf("committing the transaction: %v", err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// Return the results.
		return listResourcePermissionsOk(perms)
	}
}
