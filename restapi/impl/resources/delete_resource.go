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

// BuildDeleteResourceHandler builds the request handler for the delete resource endpoint.
func BuildDeleteResourceHandler(db *sql.DB, schema string) func(resources.DeleteResourceParams) middleware.Responder {

	// Return the handler function.
	return func(params resources.DeleteResourceParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewDeleteResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewDeleteResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that the resource exists.
		exists, err := permsdb.ResourceExists(ctx, tx, &params.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewDeleteResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if !exists {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("resource, %s, not found", params.ID)
			return resources.NewDeleteResourceNotFound().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Delete the resource.
		err = permsdb.DeleteResource(ctx, tx, &params.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewDeleteResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewDeleteResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return resources.NewDeleteResourceOK()
	}
}
