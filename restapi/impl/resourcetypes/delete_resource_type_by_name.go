package resourcetypes

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/resource_types"

	"github.com/go-openapi/runtime/middleware"
)

func deleteResourceTypeByNameOK() middleware.Responder {
	return resource_types.NewDeleteResourceTypeByNameOK()
}

func deleteResourceTypeByNameBadRequest(reason string) middleware.Responder {
	return resource_types.NewDeleteResourceTypeByNameBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func deleteResourceTypeByNameInternalServerError(reason string) middleware.Responder {
	return resource_types.NewDeleteResourceTypeByNameInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func deleteResourceTypeByNameNotFound(reason string) middleware.Responder {
	return resource_types.NewDeleteResourceTypeByNameNotFound().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildDeleteResourceTypeByNameHandler builds the request handler for the resource type by name endpoint.
func BuildDeleteResourceTypeByNameHandler(
	db *sql.DB, schema string,
) func(resource_types.DeleteResourceTypeByNameParams) middleware.Responder {

	// Return the handler function.
	return func(params resource_types.DeleteResourceTypeByNameParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}

		// Verify that the resource type exists.
		resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, &params.ResourceTypeName)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}
		if resourceType == nil {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("resource type name not found: %s", params.ResourceTypeName)
			return deleteResourceTypeByNameNotFound(reason)
		}

		// Verify that the resource type has no resources associated with it.
		numResources, err := permsdb.CountResourcesOfType(ctx, tx, resourceType.ID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}
		if numResources != 0 {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("resource type has resources associated with it: %s", params.ResourceTypeName)
			return deleteResourceTypeByNameBadRequest(reason)
		}

		// Delete the resource type.
		if err := permsdb.DeleteResourceType(ctx, tx, resourceType.ID); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteResourceTypeByNameInternalServerError(err.Error())
		}

		return deleteResourceTypeByNameOK()
	}
}
