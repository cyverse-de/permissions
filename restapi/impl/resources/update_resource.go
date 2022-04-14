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

// BuildUpdateResourceHandler builds the request handler for the update resource endpoint.
func BuildUpdateResourceHandler(db *sql.DB, schema string) func(resources.UpdateResourceParams) middleware.Responder {

	// Return the handler function.
	return func(params resources.UpdateResourceParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		resourceUpdate := params.ResourceUpdate

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that the resource exists.
		exists, err := permsdb.ResourceExists(tx, &params.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if !exists {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("resource, %s, not found", params.ID)
			return resources.NewUpdateResourceNotFound().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that another resource with the same name doesn't already exist.
		duplicate, err := permsdb.GetDuplicateResourceByName(tx, &params.ID, resourceUpdate.Name)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if duplicate != nil {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("a resource of the same type named, '%s', already exists", *resourceUpdate.Name)
			return resources.NewUpdateResourceBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Update the resource.
		resourceOut, err := permsdb.UpdateResource(tx, &params.ID, resourceUpdate.Name)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewUpdateResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return resources.NewUpdateResourceOK().WithPayload(resourceOut)
	}
}
