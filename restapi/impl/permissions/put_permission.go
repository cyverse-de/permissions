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

func putPermissionInternalServerError(reason string) middleware.Responder {
	return permissions.NewPutPermissionInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func putPermissionBadRequest(reason string) middleware.Responder {
	return permissions.NewPutPermissionBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildPutPermissionHandler builds the request handler for the put permission endpoint.
func BuildPutPermissionHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.PutPermissionParams) middleware.Responder {

	erf := &ErrorResponseFns{
		InternalServerError: putPermissionInternalServerError,
		BadRequest:          putPermissionBadRequest,
	}

	// Return the handler function.
	return func(params permissions.PutPermissionParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		req := params.Permission

		// Create a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return putPermissionInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return putPermissionInternalServerError(err.Error())
		}

		// Either get or add the subject.
		subjectID := models.ExternalSubjectID(params.SubjectID)
		subjectType := models.SubjectType(params.SubjectType)
		subjectIn := &models.SubjectIn{
			SubjectID:   &subjectID,
			SubjectType: &subjectType,
		}
		subject, errorResponder := getOrAddSubject(ctx, tx, subjectIn, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Either get or add the resource.
		resourceIn := &models.ResourceIn{
			Name:         &params.ResourceName,
			ResourceType: &params.ResourceType,
		}
		resource, errorResponder := getOrAddResource(ctx, tx, resourceIn, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Look up the permission level.
		permissionLevelID, errorResponder := getPermissionLevel(tx, *req.PermissionLevel, erf)
		if errorResponder != nil {
			tx.Rollback() // nolint:errcheck
			return errorResponder
		}

		// Either update or add the permission.
		permission, err := permsdb.UpsertPermission(tx, *subject.ID, *resource.ID, *permissionLevelID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return putPermissionInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return putPermissionInternalServerError(err.Error())
		}

		// Add the subject source ID to the permission listing.
		if err := grouperClient.AddSourceIDToPermission(permission); err != nil {
			return putPermissionInternalServerError(err.Error())
		}

		return permissions.NewPutPermissionOK().WithPayload(permission)
	}
}
