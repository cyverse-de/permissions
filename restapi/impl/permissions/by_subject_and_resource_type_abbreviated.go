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

func bySubjectAndResourceTypeAbbreviatedOk(perms []*models.AbbreviatedPermission) middleware.Responder {
	return permissions.NewBySubjectAndResourceTypeAbbreviatedOK().WithPayload(
		&models.AbbreviatedPermissionList{Permissions: perms},
	)
}

func bySubjectAndResourceTypeAbbreviatedInternalServerError(reason string) middleware.Responder {
	return permissions.NewBySubjectAndResourceTypeAbbreviatedInternalServerError().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func bySubjectAndResourceTypeAbbreviatedBadRequest(reason string) middleware.Responder {
	return permissions.NewBySubjectAndResourceTypeAbbreviatedBadRequest().WithPayload(
		&models.ErrorOut{Reason: &reason},
	)
}

func BuildBySubjectAndResourceTypeAbbreviatedHandler(
	db *sql.DB, grouperClient grouper.Grouper, schema string,
) func(permissions.BySubjectAndResourceTypeAbbreviatedParams) middleware.Responder {

	// Return the handler function.
	return func(params permissions.BySubjectAndResourceTypeAbbreviatedParams) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		subjectType := params.SubjectType
		subjectID := params.SubjectID
		resourceTypeName := params.ResourceType
		lookup := extractLookupFlag(params.Lookup)
		minLevel := params.MinLevel

		// Create a transaction for the request.
		tx, err := db.Begin()
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}
		defer tx.Rollback() // nolint:errcheck

		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}

		// Verify that the subject type is correct.
		subject, err := permsdb.GetSubjectByExternalID(ctx, tx, models.ExternalSubjectID(subjectID))
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}
		if subject != nil && string(*subject.SubjectType) != subjectType {
			reason := fmt.Sprintf("incorrect type for subject, %s: %s", subjectID, subjectType)
			return bySubjectAndResourceTypeAbbreviatedBadRequest(reason)
		}

		// Verify that the resource type exists.
		resourceType, err := permsdb.GetResourceTypeByName(ctx, tx, &resourceTypeName)
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}
		if resourceType == nil {
			return bySubjectAndResourceTypeAbbreviatedOk(make([]*models.AbbreviatedPermission, 0))
		}

		// Get the list of subject IDs to use for the query.
		subjectIDs, err := buildSubjectIDList(grouperClient, subjectType, subjectID, lookup)
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}

		// Perform the lookup.
		perms, err := permsdb.AbbreviatedPermissionsForSubjectAndResourceType(
			tx, subjectIDs, resourceTypeName, minLevel,
		)
		if err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}

		// Commit the transaction.
		if err := tx.Commit(); err != nil {
			logger.Log.Error(err)
			return bySubjectAndResourceTypeAbbreviatedInternalServerError(err.Error())
		}

		return bySubjectAndResourceTypeAbbreviatedOk(perms)
	}
}
