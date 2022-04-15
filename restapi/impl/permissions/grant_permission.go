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

func grantPermissionInternalServerError(reason string) middleware.Responder {
	return permissions.NewGrantPermissionInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func grantPermissionBadRequest(reason string) middleware.Responder {
	return permissions.NewGrantPermissionBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildGrantPermissionHandler builds the request handler for the grant permissions endpoint.
func BuildGrantPermissionHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.GrantPermissionParams) middleware.Responder {

	erf := &ErrorResponseFns{
		InternalServerError: grantPermissionInternalServerError,
		BadRequest:          grantPermissionBadRequest,
	}

	// Return the hnadler function.
	return func(params permissions.GrantPermissionParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		req := params.PermissionGrantRequest

		// Create a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return grantPermissionInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return grantPermissionInternalServerError(err.Error())
		}

		// Either get or add the subject.
		subject, errorResponder := getOrAddSubject(ctx, tx, req.Subject, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Either get or add the resource.
		resource, errorResponder := getOrAddResource(ctx, tx, req.Resource, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Look up the permission level.
		permissionLevelID, errorResponder := getPermissionLevel(ctx, tx, *req.PermissionLevel, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Either update or add the permission.
		permission, err := permsdb.UpsertPermission(ctx, tx, *subject.ID, *resource.ID, *permissionLevelID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return grantPermissionInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return grantPermissionInternalServerError(err.Error())
		}

		// Add the subject source ID to the permission object.
		if err := grouperClient.AddSourceIDToPermission(ctx, permission); err != nil {
			logger.Log.Error(err)
			return grantPermissionInternalServerError(err.Error())
		}

		return permissions.NewGrantPermissionOK().WithPayload(permission)
	}
}
