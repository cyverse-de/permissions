package permissions

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"

	"github.com/go-openapi/runtime/middleware"
)

func revokePermissionInternalServerError(reason string) middleware.Responder {
	return permissions.NewRevokePermissionInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func revokePermissionNotFound(reason string) middleware.Responder {
	return permissions.NewRevokePermissionNotFound().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildRevokePermissionHandler builds the request handler for the revoke permission endpoint.
func BuildRevokePermissionHandler(db *sql.DB, schema string) func(permissions.RevokePermissionParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.RevokePermissionParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Create a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}

		// Look up the resource type.
		resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, &params.ResourceType)
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}
		if resourceType == nil {
			reason := fmt.Sprintf("resource type not found: %s", params.ResourceType)
			return revokePermissionNotFound(reason)
		}

		// Look up the resource.
		resource, err := permsdb.GetResourceByName(ctx, tx, &params.ResourceName, resourceType.ID)
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}
		if resource == nil {
			reason := fmt.Sprintf("resource not found: %s/%s", params.ResourceType, params.ResourceName)
			return revokePermissionNotFound(reason)
		}

		// Look up the subject.
		subjectType := models.SubjectType(params.SubjectType)
		subjectID := models.ExternalSubjectID(params.SubjectID)
		subject, err := permsdb.GetSubject(ctx, tx, subjectID, subjectType)
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}
		if subject == nil {
			reason := fmt.Sprintf("subject not found: %s/%s", subjectType, subjectID)
			return revokePermissionNotFound(reason)
		}

		// Look up the permission.
		permission, err := permsdb.GetPermission(ctx, tx, *subject.ID, *resource.ID)
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}
		if permission == nil {
			reason := fmt.Sprintf(
				"permission not found: %s/%s:%s/%s", params.ResourceType, params.ResourceName, subjectType, subjectID,
			)
			return revokePermissionNotFound(reason)
		}

		// Delete the permission.
		err = permsdb.DeletePermission(ctx, tx, *permission.ID)
		if err != nil {
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return revokePermissionInternalServerError(err.Error())
		}

		return permissions.NewRevokePermissionOK()
	}
}
