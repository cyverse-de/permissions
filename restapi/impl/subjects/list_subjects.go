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

func listSubjectsInternalServerError(reason string) middleware.Responder {
	return subjects.NewListSubjectsInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildListSubjectsHandler builds the request handler for the list subjects endpoint.
func BuildListSubjectsHandler(db *sql.DB, schema string) func(subjects.ListSubjectsParams) middleware.Responder {

	// Return the handler function.
	return func(params subjects.ListSubjectsParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()

		// Start a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return listSubjectsInternalServerError(err.Error())
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return listSubjectsInternalServerError(err.Error())
		}

		// Obtain the list of subjects.
		result, err := permsdb.ListSubjects(ctx, tx, params.SubjectType, params.SubjectID)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return listSubjectsInternalServerError(err.Error())
		}

		// Commit the transaction for the request.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return listSubjectsInternalServerError(err.Error())
		}

		// Return the result.
		return subjects.NewListSubjectsOK().WithPayload(&models.SubjectsOut{Subjects: result})
	}
}
