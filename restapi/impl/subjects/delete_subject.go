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

// BuildDeleteSubjectHandler builds the request handler for the delete subject endpoint.
func BuildDeleteSubjectHandler(db *sql.DB, schema string) func(subjects.DeleteSubjectParams) middleware.Responder {

	// Return the handler function.
	return func(params subjects.DeleteSubjectParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		id := models.InternalSubjectID(params.ID)

		// Start a transaction for this request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewDeleteSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewDeleteSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Verify that the subject exists.
		exists, err := permsdb.SubjectExists(ctx, tx, id)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewDeleteSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}
		if !exists {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("subject, %s, not found", string(id))
			return subjects.NewDeleteSubjectNotFound().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Delete the subject.
		if err := permsdb.DeleteSubject(ctx, tx, id); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewDeleteSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			reason := err.Error()
			return subjects.NewDeleteSubjectInternalServerError().WithPayload(
				&models.ErrorOut{Reason: &reason},
			)
		}

		return subjects.NewDeleteSubjectOK()
	}
}
