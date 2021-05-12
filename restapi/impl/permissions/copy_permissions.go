package permissions

import (
	"database/sql"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"
	"github.com/go-openapi/runtime/middleware"
)

func copyPermissionsOk() middleware.Responder {
	return permissions.NewCopyPermissionsOK()
}

func copyPermissionsBadRequest(reason string) middleware.Responder {
	return permissions.NewCopyPermissionsBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func copyPermissionsInternalServerError(reason string) middleware.Responder {
	return permissions.NewCopyPermissionsInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildCopyPermissionsHandler builds the request handler for the copy permissions endpoint.
func BuildCopyPermissionsHandler(db *sql.DB) func(permissions.CopyPermissionsParams) middleware.Responder {

	erf := &ErrorResponseFns{
		InternalServerError: copyPermissionsInternalServerError,
		BadRequest:          copyPermissionsBadRequest,
	}

	// Return the handler function.
	return func(params permissions.CopyPermissionsParams) middleware.Responder {
		sourceType := models.SubjectType(params.SubjectType)
		sourceID := models.ExternalSubjectID(params.SubjectID)
		destSubjects := params.DestSubjects.Subjects

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return copyPermissionsInternalServerError(err.Error())
		}

		// Either get or add the source subject.
		source, errorResponse := getOrAddSubject(tx, &models.SubjectIn{SubjectType: &sourceType, SubjectID: &sourceID}, erf)
		if errorResponse != nil {
			tx.Rollback()
			return errorResponse
		}

		// Copy the source subject's permissions to each destination subject.
		for _, destIn := range destSubjects {

			// Either get or add the subject.
			dest, errorResponse := getOrAddSubject(tx, destIn, erf)
			if errorResponse != nil {
				tx.Rollback()
				return errorResponse
			}

			// Copy the permissions.
			if err := permsdb.CopyPermissions(tx, source, dest); err != nil {
				tx.Rollback()
				logger.Log.Error(err)
				return copyPermissionsInternalServerError(err.Error())
			}
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			logger.Log.Error(err)
			return copyPermissionsInternalServerError(err.Error())
		}

		return copyPermissionsOk()
	}
}
