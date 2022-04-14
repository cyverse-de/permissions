package permissions

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/go-openapi/runtime/middleware"
)

func internalServerError(reason string) *permissions.ListPermissionsInternalServerError {
	return permissions.NewListPermissionsInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildListPermissionsHandler builds the request handler for the list permissions endpoint.
func BuildListPermissionsHandler(
	db *sql.DB, grouper grouper.Grouper, schema string,
) func(permissions.ListPermissionsParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.ListPermissionsParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return internalServerError(err.Error())
		}
		defer tx.Commit() // nolint:errcheck

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return internalServerError(err.Error())
		}

		// List all permissions.
		result, err := permsdb.ListPermissions(tx)
		if err != nil {
			logger.Log.Error(err)
			return internalServerError(err.Error())
		}

		// Add subject sources to the permission list.
		if err = grouper.AddSourceIDToPermissions(result); err != nil {
			logger.Log.Error(err)
			return internalServerError(err.Error())
		}

		// Return the results.
		return permissions.NewListPermissionsOK().WithPayload(&models.PermissionList{Permissions: result})
	}
}
