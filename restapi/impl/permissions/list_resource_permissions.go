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

// BuildListResourcePermissionsHandler builds the request handler for the list resource permissions endpoint.
func BuildListResourcePermissionsHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.ListResourcePermissionsParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.ListResourcePermissionsParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		resourceTypeName := params.ResourceType
		resourceName := params.ResourceName

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// List the permissions for the resource.
		perms, err := permsdb.ListResourcePermissions(ctx, tx, resourceTypeName, resourceName)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// Add the subject source ID to the response body.
		if err := grouperClient.AddSourceIDToPermissions(perms); err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
		}

		// Return the results.
		return listResourcePermissionsOk(perms)
	}
}
