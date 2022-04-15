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

// BuildAddResourceHandler builds the request handler for the add resource endpoint.
func BuildAddResourceHandler(db *sql.DB, schema string) func(resources.AddResourceParams) middleware.Responder {

	// Return the handler function.
	return func(params resources.AddResourceParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		resourceIn := params.ResourceIn

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Load the resource type.
		resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, resourceIn.ResourceType)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if resourceType == nil {
			reason := fmt.Sprintf("no resource type named, '%s', found", *resourceIn.ResourceType)
			return resources.NewAddResourceBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that another resource with the same name doesn't already exist.
		duplicate, err := permsdb.GetResourceByName(tx, resourceIn.Name, resourceType.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if duplicate != nil {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf(
				"a resource named, '%s', with type, '%s', already exists", *resourceIn.Name, *resourceType.Name,
			)
			return resources.NewAddResourceBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Add the resource to the database.
		resourceOut, err := permsdb.AddResource(tx, resourceIn.Name, resourceType.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return resources.NewAddResourceInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return resources.NewAddResourceCreated().WithPayload(resourceOut)
	}
}
