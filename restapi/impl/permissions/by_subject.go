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

func bySubjectOk(perms []*models.Permission) middleware.Responder {
	return permissions.NewBySubjectOK().WithPayload(
		&models.PermissionList{Permissions: perms},
	)
}

func bySubjectInternalServerError(reason string) middleware.Responder {
	return permissions.NewBySubjectInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func bySubjectBadRequest(reason string) middleware.Responder {
	return permissions.NewBySubjectBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

// BuildBySubjectHandler builds the request handler for the permissions by subject endpoint
func BuildBySubjectHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.BySubjectParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.BySubjectParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		subjectType := params.SubjectType
		subjectID := params.SubjectID
		lookup := extractLookupFlag(params.Lookup)
		minLevel := params.MinLevel

		// Create a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		// Verify that the subject type is correct.
		subject, err := permsdb.GetSubjectByExternalID(ctx, tx, models.ExternalSubjectID(subjectID))
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}
		if subject != nil && string(*subject.SubjectType) != subjectType {
			tx.Rollback() // nolint:errcheck
			reason := fmt.Sprintf("incorrect type for subject, %s: %s", subjectID, subjectType)
			return bySubjectBadRequest(reason)
		}

		// Get the list of subject IDs to use for the query.
		subjectIds, err := buildSubjectIDList(grouperClient, subjectType, subjectID, lookup)
		if err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		// Perform the lookup.
		var perms []*models.Permission
		if minLevel == nil {
			perms, err = permsdb.PermissionsForSubjects(tx, subjectIds)
			if err != nil {
				tx.Rollback() // nolint:errcheck
				logger.Log.Error(err)
				return bySubjectInternalServerError(err.Error())
			}
		} else {
			perms, err = permsdb.PermissionsForSubjectsMinLevel(tx, subjectIds, *minLevel)
			if err != nil {
				tx.Rollback() // nolint:errcheck
				logger.Log.Error(err)
				return bySubjectInternalServerError(err.Error())
			}
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			tx.Rollback() // nolint:errcheck
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		// Add the subject source ID to the response body.
		if err := grouperClient.AddSourceIDToPermissions(perms); err != nil {
			logger.Log.Error(err)
			return bySubjectInternalServerError(err.Error())
		}

		return bySubjectOk(perms)
	}
}
