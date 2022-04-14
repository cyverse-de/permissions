package subjects

import (
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	"github.com/cyverse-de/permissions/restapi/operations/subjects"

	"github.com/go-openapi/runtime/middleware"
)

func deleteSubjectByExternalIDInternalServerError(reason string) middleware.Responder {
	return subjects.NewDeleteSubjectByExternalIDInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func deleteSubjectByExternalIDNotFound(reason string) middleware.Responder {
	return subjects.NewDeleteSubjectByExternalIDNotFound().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func deleteSubjectByExternalIDOk() middleware.Responder {
	return subjects.NewDeleteSubjectByExternalIDOK()
}

// BuildDeleteSubjectByExternalIDHandler builds the request handler for the delete subject by external ID endpoint.
func BuildDeleteSubjectByExternalIDHandler(
	db *sql.DB, schema string,
) func(subjects.DeleteSubjectByExternalIDParams) middleware.Responder {

	// Return the handler function.
	return func(params subjects.DeleteSubjectByExternalIDParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		subjectID := models.ExternalSubjectID(params.SubjectID)
		subjectType := models.SubjectType(params.SubjectType)

		// Start a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return deleteSubjectByExternalIDInternalServerError(err.Error())
		}

		_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return deleteSubjectByExternalIDInternalServerError(err.Error())
		}

		// Look up the subject.
		subject, err := permsdb.GetSubject(ctx, tx, subjectID, subjectType)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteSubjectByExternalIDInternalServerError(err.Error())
		}
		if subject == nil {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("subject not found: %s:%s", string(subjectType), string(subjectID))
			return deleteSubjectByExternalIDNotFound(reason)
		}

		// Delete the subject.
		if err := permsdb.DeleteSubject(ctx, tx, *subject.ID); err != nil {
			tx.Rollback() // nolint:errcheck
			return deleteSubjectByExternalIDInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return deleteSubjectByExternalIDInternalServerError(err.Error())
		}

		return deleteSubjectByExternalIDOk()
	}
}
