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
	db *sql.DB, schema string, maxExpandedSubjects int,
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

		// List the permissions for the resource, expanding group permissions to
		// their members when asked.
		var perms []*models.Permission
		if expandGroups {
			// NOTE: an ErrTooManySubjects refusal deserves a 4xx -- the request
			// is well formed and the server is declining to materialize an
			// unbounded response -- but the generated contract for this endpoint
			// offers only 200 and 500. Adding one means changing swagger.yml and
			// regenerating, so for now it falls through to the generic 500.
			perms, err = permsdb.ExpandedResourcePermissions(
				ctx, tx, resourceTypeName, resourceName, maxExpandedSubjects)
		} else {
			perms, err = permsdb.ListResourcePermissions(ctx, tx, resourceTypeName, resourceName)
		}
		if err != nil {
			logger.Log.Error(err)
			return listResourcePermissionsInternalServerError(err.Error())
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
