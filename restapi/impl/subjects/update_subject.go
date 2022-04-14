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

// BuildUpdateSubjectHandler builds the request handler for the update subject endpoint.
func BuildUpdateSubjectHandler(db *sql.DB, schema string) func(subjects.UpdateSubjectParams) middleware.Responder {

	// Return the handler function.
	return func(params subjects.UpdateSubjectParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		id := models.InternalSubjectID(params.ID)
		subjectIn := params.SubjectIn

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that the subject exists.
		exists, err := permsdb.SubjectExists(ctx, tx, id)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if !exists {
			reason := fmt.Sprintf("subject, %s, not found", string(id))
			return subjects.NewUpdateSubjectNotFound().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that a subject with the same external subject ID doesn't exist.
		duplicateExists, err := permsdb.DuplicateSubjectExists(ctx, tx, id, *subjectIn.SubjectID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if duplicateExists {
			reason := fmt.Sprintf("another subject with the ID, %s, already exists", string(*subjectIn.SubjectID))
			return subjects.NewUpdateSubjectBadRequest().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Update the subject.
		subjectOut, err := permsdb.UpdateSubject(ctx, tx, id, *subjectIn.SubjectID, *subjectIn.SubjectType)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewUpdateSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return subjects.NewUpdateSubjectOK().WithPayload(subjectOut)
	}
}
