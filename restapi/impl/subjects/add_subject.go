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

// BuildAddSubjectHandler builds the request handler for the add subject endpoint.
func BuildAddSubjectHandler(db *sql.DB, schema string) func(subjects.AddSubjectParams) middleware.Responder {

	// Return the handler function.
	return func(params subjects.AddSubjectParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		subjectIn := params.SubjectIn

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewAddSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewAddSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Make sure that a subject with the same ID doesn't exist already.
		exists, err := permsdb.SubjectIDExists(ctx, tx, *subjectIn.SubjectID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewAddSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if exists {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("subject, %s, already exists", string(*subjectIn.SubjectID))
			return subjects.NewAddSubjectBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Add the subject.
		subjectOut, err := permsdb.AddSubject(ctx, tx, *subjectIn.SubjectID, *subjectIn.SubjectType)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewAddSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewAddSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return subjects.NewAddSubjectCreated().WithPayload(subjectOut)
	}
}
