package resources

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/resources"

	"github.com/go-openapi/runtime/middleware"
)

// BuildListResourcesHandler builds the request handler for the list resources endpoint.
func BuildListResourcesHandler(db *sql.DB, schema string) func(resources.ListResourcesParams) middleware.Responder {

	// Return the handler function.
	return func(params resources.ListResourcesParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewListResourcesInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		defer tx.Commit() // nolint:errcheck

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewListResourcesInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// List all resources.
		result, err := permsdb.ListResources(tx, params.ResourceTypeName, params.ResourceName)
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewListResourcesInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Return the results.
		return resources.NewListResourcesOK().WithPayload(&models.ResourcesOut{Resources: result})
	}
}
