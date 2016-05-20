package permissions

import (
	"database/sql"
	"fmt"
	"github.com/go-swagger/go-swagger/httpkit/middleware"
	"logcabin"
	"permissions/models"
	permsdb "permissions/restapi/impl/db"
)

type ErrorResponseFns struct {
	BadRequest          func(string) middleware.Responder
	InternalServerError func(string) middleware.Responder
}

func getOrAddSubject(
	tx *sql.Tx,
	subjectIn *models.SubjectIn,
	erf *ErrorResponseFns,
) (*models.SubjectOut, middleware.Responder) {

	// Attempt to look up the subject.
	subject, err := permsdb.GetSubject(tx, subjectIn.SubjectID, subjectIn.SubjectType)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if subject != nil {
		return subject, nil
	}

	// Make sure that another subject with the same ID doesn't exist already.
	exists, err := permsdb.SubjectIdExists(tx, subjectIn.SubjectID)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError((err.Error()))
	}
	if exists {
		reason := fmt.Sprintf("another subject with ID, %s, already exists", string(subjectIn.SubjectID))
		return nil, erf.BadRequest(reason)
	}

	// Attempt to add the subject.
	subject, err = permsdb.AddSubject(tx, subjectIn.SubjectID, subjectIn.SubjectType)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	return subject, nil
}

func getOrAddResource(
	tx *sql.Tx,
	resourceIn *models.ResourceIn,
	erf *ErrorResponseFns,
) (*models.ResourceOut, middleware.Responder) {

	// Look up the resource type.
	resourceType, err := permsdb.GetResourceTypeByName(tx, resourceIn.ResourceType)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if resourceType == nil {
		reason := fmt.Sprintf("no resource type named, %s, found", *resourceIn.ResourceType)
		return nil, erf.BadRequest(reason)
	}

	// Attempt to look up the resource.
	resource, err := permsdb.GetResourceByName(tx, resourceIn.Name, resourceType.ID)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if resource != nil {
		return resource, nil
	}

	// Attempt to add the resource.
	resource, err = permsdb.AddResource(tx, resourceIn.Name, resourceType.ID)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	return resource, nil
}

func getPermissionLevel(
	tx *sql.Tx,
	level models.PermissionLevel,
	erf *ErrorResponseFns,
) (*string, middleware.Responder) {

	// Look up the permission level.
	permissionLevelId, err := permsdb.GetPermissionLevelIdByName(tx, level)
	if err != nil {
		logcabin.Error.Print(err)
		return nil, erf.InternalServerError(err.Error())
	}
	if permissionLevelId == nil {
		reason := fmt.Sprintf("no permission level named, %s, found", string(level))
		return nil, erf.BadRequest(reason)
	}

	return permissionLevelId, nil
}
