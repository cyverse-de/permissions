package permissions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"

	"github.com/go-openapi/runtime/middleware"
)

// ErrorResponseFns is a structure containing functions that can be used to generate responses for erroneous requests.
type ErrorResponseFns struct {
	BadRequest          func(string) middleware.Responder
	InternalServerError func(string) middleware.Responder
}

func getOrAddSubject(
	ctx context.Context,
	tx *sql.Tx,
	subjectIn *models.SubjectIn,
	erf *ErrorResponseFns,
) (*models.SubjectOut, middleware.Responder) {

	// Attempt to look up the subject.
	subject, err := permsdb.GetSubject(ctx, tx, *subjectIn.SubjectID, *subjectIn.SubjectType)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if subject != nil {
		return subject, nil
	}

	// Make sure that another subject with the same ID doesn't exist already.
	exists, err := permsdb.SubjectIDExists(ctx, tx, *subjectIn.SubjectID)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError((err.Error()))
	}
	if exists {
		reason := fmt.Sprintf("another subject with ID, %s, already exists", string(*subjectIn.SubjectID))
		return nil, erf.BadRequest(reason)
	}

	// Attempt to add the subject.
	subject, err = permsdb.AddSubject(ctx, tx, *subjectIn.SubjectID, *subjectIn.SubjectType)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	return subject, nil
}

func getOrAddResource(
	ctx context.Context,
	tx *sql.Tx,
	resourceIn *models.ResourceIn,
	erf *ErrorResponseFns,
) (*models.ResourceOut, middleware.Responder) {

	// Look up the resource type.
	resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, resourceIn.ResourceType)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if resourceType == nil {
		reason := fmt.Sprintf("no resource type named, %s, found", *resourceIn.ResourceType)
		return nil, erf.BadRequest(reason)
	}

	// Attempt to look up the resource.
	resource, err := permsdb.GetResourceByName(ctx, tx, resourceIn.Name, resourceType.ID)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if resource != nil {
		return resource, nil
	}

	// Attempt to add the resource.
	resource, err = permsdb.AddResource(ctx, tx, resourceIn.Name, resourceType.ID)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	return resource, nil
}

func getPermissionLevel(
	ctx context.Context,
	tx *sql.Tx,
	level models.PermissionLevel,
	erf *ErrorResponseFns,
) (*string, middleware.Responder) {

	// Look up the permission level.
	permissionLevelID, err := permsdb.GetPermissionLevelIDByName(ctx, tx, level)
	if err != nil {
		logger.Log.Error(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if permissionLevelID == nil {
		reason := fmt.Sprintf("no permission level named, %s, found", string(level))
		return nil, erf.BadRequest(reason)
	}

	return permissionLevelID, nil
}

func extractLookupFlag(lookup *bool) bool {
	if lookup != nil {
		return *lookup
	}
	return false
}

func groupIdsForSubject(grouperClient grouper.Grouper, subjectType, subjectID string) ([]string, error) {
	groupIds := make([]string, 0)

	// Simply return an empty slice if the subject is a group.
	if subjectType == "group" {
		return groupIds, nil
	}

	// Look up the groups.
	groups, err := grouperClient.GroupsForSubject(subjectID)
	if err != nil {
		return nil, err
	}

	// Extract the identifiers from the list of groups.
	for _, group := range groups {
		groupIds = append(groupIds, group.ID)
	}

	return groupIds, nil
}

func buildSubjectIDList(grouperClient grouper.Grouper, subjectType, subjectID string, lookup bool) ([]string, error) {
	if lookup {
		groupIds, err := groupIdsForSubject(grouperClient, subjectType, subjectID)
		if err != nil {
			return nil, err
		}
		return append(groupIds, subjectID), nil
	}
	return []string{subjectID}, nil
}
