package permissions

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"

	"github.com/go-openapi/runtime/middleware"
)

func bySubjectAndResourceOk(perms []*models.Permission) middleware.Responder {
	return permissions.NewBySubjectAndResourceOK().WithPayload(
		&models.PermissionList{Permissions: perms},
	)
}

func bySubjectAndResourceInternalServerError(reason string) middleware.Responder {
	return permissions.NewBySubjectAndResourceInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func bySubjectAndResourceBadRequest(reason string) middleware.Responder {
	return permissions.NewBySubjectAndResourceBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildBySubjectAndResourceHandler builds the request handler for the permissions by subject and resource endpoint.
func BuildBySubjectAndResourceHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.BySubjectAndResourceParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.BySubjectAndResourceParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		subjectType := params.SubjectType
		subjectID := params.SubjectID
		resourceTypeName := params.ResourceType
		resourceName := params.ResourceName
		lookup := extractLookupFlag(params.Lookup)
		minLevel := params.MinLevel

		// Start a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}

		// Verify that the subject type is correct.
		subject, err := permsdb.GetSubjectByExternalID(ctx, tx, models.ExternalSubjectID(subjectID))
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}
		if subject != nil && string(*subject.SubjectType) != subjectType {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("incorrect type for subject, %s: %s", subjectID, subjectType)
			return bySubjectAndResourceBadRequest(reason)
		}

		// Verify that the resource type exists.
		resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, &resourceTypeName)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}
		if resourceType == nil {
			tx.Rollback() // nolint:errcheck
			return bySubjectAndResourceOk(make([]*models.Permission, 0))
		}

		// Verify that the resource exists.
		resource, err := permsdb.GetResourceByName(ctx, tx, &resourceName, resourceType.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}
		if resource == nil {
			tx.Rollback() // nolint:errcheck
			return bySubjectAndResourceOk(make([]*models.Permission, 0))
		}

		// Get the list of subject IDs to use for the query.
		subjectIds, err := buildSubjectIDList(ctx, grouperClient, subjectType, subjectID, lookup)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		// Perform the lookup.
		var perms []*models.Permission
		if minLevel == nil {
			perms, err = permsdb.PermissionsForSubjectsAndResource(ctx, tx, subjectIds, resourceTypeName, resourceName)
			if err != nil {
				tx.Rollback() // nolint:errcheck
				logger.Log.Error(err)
				return bySubjectAndResourceInternalServerError(err.Error())
			}
		} else {
			perms, err = permsdb.PermissionsForSubjectsAndResourceMinLevel(
				ctx, tx, subjectIds, resourceTypeName, resourceName, *minLevel,
			)
			if err != nil {
				tx.Rollback() // nolint:errcheck
				logger.Log.Error(err)
				return bySubjectAndResourceInternalServerError(err.Error())
			}
		}

		// Commit the transaction.
		err = tx.Commit()
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}

		// Add the subject source ID to the results.
		if err := grouperClient.AddSourceIDToPermissions(ctx, perms); err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceInternalServerError(err.Error())
		}

		return bySubjectAndResourceOk(perms)
	}
}
