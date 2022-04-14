package resourcetypes

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/resource_types"
	"github.com/go-openapi/runtime/middleware"
)

func buildResourceTypesGetResponse(
	db *sql.DB, schema string, params resource_types.GetResourceTypesParams,
) (*models.ResourceTypesOut, error) {
	ctx := params.HTTPRequest.Context()

	resourceTypeName := params.ResourceTypeName

	// Start a transaction for the request.
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit() // nolint:errcheck

	_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
	if err != nil {
		return nil, err
	}

	// Get the list of resource types.
	resourceTypes, err := permsdb.ListResourceTypes(tx, resourceTypeName)
	if err != nil {
		return nil, err
	}

	return &models.ResourceTypesOut{ResourceTypes: resourceTypes}, nil
}

// BuildResourceTypesGetHandler builds the request handler for the resource type listing endpoint.
func BuildResourceTypesGetHandler(db *sql.DB, schema string) func(resource_types.GetResourceTypesParams) middleware.Responder {

	// Return the handler function.
	return func(params resource_types.GetResourceTypesParams) middleware.Responder {
		response, err := buildResourceTypesGetResponse(db, schema, params)
		if err != nil {
			reason := err.Error()
			return resource_types.NewGetResourceTypesInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		return resource_types.NewGetResourceTypesOK().WithPayload(response)
	}
}
