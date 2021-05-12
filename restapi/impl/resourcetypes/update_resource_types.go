package resourcetypes

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/resource_types"
	"github.com/go-openapi/runtime/middleware"
)

// BuildResourceTypesIDPutHandler builds the request handler for the update resource type endpoint.
func BuildResourceTypesIDPutHandler(db *sql.DB) func(resource_types.PutResourceTypesIDParams) middleware.Responder {

	// Return the handler function.
	return func(params resource_types.PutResourceTypesIDParams) middleware.Responder {
		resourceTypeIn := params.ResourceTypeIn

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			reason := err.Error()
			return resource_types.NewPutResourceTypesIDInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that the resource type exists.
		exists, err := permsdb.ResourceTypeExists(tx, &params.ID)
		if err != nil {
			tx.Rollback()
			reason := err.Error()
			return resource_types.NewPutResourceTypesIDInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if !exists {
			tx.Rollback()
			reason := fmt.Sprintf("resource type %s not found", params.ID)
			return resource_types.NewPutResourceTypesIDNotFound().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Check for a duplicate name.
		duplicate, err := permsdb.GetDuplicateResourceTypeByName(tx, &params.ID, resourceTypeIn.Name)
		if err != nil {
			tx.Rollback()
			reason := err.Error()
			return resource_types.NewPutResourceTypesIDInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if duplicate != nil {
			tx.Rollback()
			reason := fmt.Sprintf("another resource type named %s already exists", *resourceTypeIn.Name)
			return resource_types.NewPutResourceTypesIDBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Update the resource type.
		resourceTypeOut, err := permsdb.UpdateResourceType(tx, &params.ID, resourceTypeIn)
		if err != nil {
			tx.Rollback()
			reason := err.Error()
			return resource_types.NewPutResourceTypesIDInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			reason := err.Error()
			return resource_types.NewPutResourceTypesIDInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return resource_types.NewPutResourceTypesIDOK().WithPayload(resourceTypeOut)
	}
}
