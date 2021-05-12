package test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/cyverse-de/permissions/models"
	"github.com/cyverse-de/permissions/restapi/operations/subjects"

	impl "github.com/cyverse-de/permissions/restapi/impl/subjects"
	middleware "github.com/go-openapi/runtime/middleware"
)

func checkSubject(t *testing.T, subjects []*models.SubjectOut, i int32, subjectId, subjectType string) {
	actual := subjects[i]
	if *actual.SubjectID != models.ExternalSubjectID(subjectId) {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if *actual.SubjectType != models.SubjectType(subjectType) {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func addSubjectAttempt(
	db *sql.DB,
	subjectId models.ExternalSubjectID,
	subjectType models.SubjectType,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildAddSubjectHandler(db)

	// Attempt to add the subject to the database.
	subjectIn := &models.SubjectIn{SubjectID: &subjectId, SubjectType: &subjectType}
	params := subjects.AddSubjectParams{SubjectIn: subjectIn}
	return handler(params)
}

func addSubject(db *sql.DB, subjectId models.ExternalSubjectID, subjectType models.SubjectType) *models.SubjectOut {
	responder := addSubjectAttempt(db, subjectId, subjectType)
	return responder.(*subjects.AddSubjectCreated).Payload
}

func listSubjectsAttempt(db *sql.DB, subjectType, subjectId *string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildListSubjectsHandler(db)

	// Attempt to list the subjects.
	params := subjects.ListSubjectsParams{SubjectType: subjectType, SubjectID: subjectId}
	return handler(params)
}

func listSubjects(db *sql.DB, subjectType, subjectId *string) *models.SubjectsOut {
	responder := listSubjectsAttempt(db, subjectType, subjectId)
	return responder.(*subjects.ListSubjectsOK).Payload
}

func updateSubjectAttempt(
	db *sql.DB,
	id models.InternalSubjectID,
	subjectId models.ExternalSubjectID,
	subjectType models.SubjectType,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildUpdateSubjectHandler(db)

	// Attempt to update the subject.
	subjectIn := &models.SubjectIn{SubjectID: &subjectId, SubjectType: &subjectType}
	params := subjects.UpdateSubjectParams{ID: string(id), SubjectIn: subjectIn}
	return handler(params)
}

func updateSubject(
	db *sql.DB,
	id models.InternalSubjectID,
	subjectId models.ExternalSubjectID,
	subjectType models.SubjectType,
) *models.SubjectOut {
	responder := updateSubjectAttempt(db, id, subjectId, subjectType)
	return responder.(*subjects.UpdateSubjectOK).Payload
}

func deleteSubjectAttempt(db *sql.DB, id models.InternalSubjectID) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteSubjectHandler(db)

	// Attempt to delete the subject.
	params := subjects.DeleteSubjectParams{ID: string(id)}
	return handler(params)
}

func deleteSubject(db *sql.DB, id models.InternalSubjectID) {
	responder := deleteSubjectAttempt(db, id)
	_ = responder.(*subjects.DeleteSubjectOK)
}

func deleteSubjectByExternalIdAttempt(db *sql.DB, subjectId, subjectType string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteSubjectByExternalIdHandler(db)

	// Attempt to delete the subject.
	params := subjects.DeleteSubjectByExternalIDParams{SubjectID: subjectId, SubjectType: subjectType}
	return handler(params)
}

func deleteSubjectByExternalId(db *sql.DB, subjectId, subjectType string) {
	responder := deleteSubjectByExternalIdAttempt(db, subjectId, subjectType)
	_ = responder.(*subjects.DeleteSubjectByExternalIDOK)
}

func TestAddSubject(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	subjectId := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	subject := addSubject(db, subjectId, subjectType)

	// Verify that we got the expected response.
	if *subject.SubjectID != subjectId {
		t.Errorf("unexpected subject ID: %s", *subject.SubjectID)
	}
	if *subject.SubjectType != subjectType {
		t.Errorf("unexpected subject type: %s", *subject.SubjectType)
	}
}

func TestAddDuplicateSubject(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	subjectId := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	addSubject(db, subjectId, subjectType)

	// Attempt to add a subject with the same ID.
	responder := addSubjectAttempt(db, subjectId, subjectType)
	errorOut := responder.(*subjects.AddSubjectBadRequest).Payload

	// Verify that we got the expected result.
	expected := fmt.Sprintf("subject, %s, already exists", string(subjectId))
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestListSubjects(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	subjectId := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	expected := addSubject(db, subjectId, subjectType)

	// List the subjects and verify that we get the expected number of results.
	subjectList := listSubjects(db, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	actual := subjectList[0]
	if expected.ID != actual.ID {
		t.Errorf("unexpected ID: %s", string(*actual.ID))
	}
	if expected.SubjectID != actual.SubjectID {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if expected.SubjectType != actual.SubjectType {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func TestListSubjectsByExternalId(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	expected := addSubject(db, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectId := "a"
	subjectList := listSubjects(db, nil, &subjectId).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	actual := subjectList[0]
	if expected.ID != actual.ID {
		t.Errorf("unexpected ID: %s", string(*actual.ID))
	}
	if expected.SubjectID != actual.SubjectID {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if expected.SubjectType != actual.SubjectType {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func TestListSubjectsByType(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	addSubject(db, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectType := "user"
	subjectList := listSubjects(db, &subjectType, nil).Subjects
	if len(subjectList) != 2 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	checkSubject(t, subjectList, 0, "a", "user")
	checkSubject(t, subjectList, 1, "c", "user")
}

func TestListSubjectsByExternalIdAndType(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject.
	addSubject(db, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectId := "a"
	subjectType := "user"
	subjectList := listSubjects(db, &subjectType, &subjectId).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	checkSubject(t, subjectList, 0, "a", "user")
}

func TestListSubjectsEmpty(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// List the subjects and verify that we get the expected result.
	subjectList := listSubjects(db, nil, nil).Subjects
	if subjectList == nil {
		t.Error("nil value returned as a subject list")
	}
	if len(subjectList) != 0 {
		t.Errorf("unexpected number of results: %d", len(subjectList))
	}
}

func TestUpdateSubject(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject to the database.
	origId := models.ExternalSubjectID("s1")
	origType := models.SubjectType("user")
	orig := addSubject(db, origId, origType)

	// Change the subject ID and type.
	newId := models.ExternalSubjectID("s2")
	newType := models.SubjectType("group")
	new := updateSubject(db, *orig.ID, newId, newType)

	// Verify that we got the expected result.
	if new.ID != orig.ID {
		t.Errorf("unexpected internal ID returned: %s", *new.ID)
	}
	if *new.SubjectID != newId {
		t.Errorf("unexpected external ID returned: %s", *new.SubjectID)
	}
	if *new.SubjectType != newType {
		t.Errorf("unexpected subject type returned: %s", *new.SubjectType)
	}

	// List the subjects and verify that we get the expected number of results.
	subjectList := listSubjects(db, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}

	// Verify that we get the expected result.
	listed := subjectList[0]
	if listed.ID != orig.ID {
		t.Errorf("unexpected internal ID listed: %s", *listed.ID)
	}
	if *listed.SubjectID != newId {
		t.Errorf("unexpected external ID listed: %s", *listed.SubjectID)
	}
	if *listed.SubjectType != newType {
		t.Errorf("unexpected subject type listed: %s", *listed.SubjectType)
	}
}

func TestUpdateSubjectNotFound(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Attempt to update a subject.
	newId := models.ExternalSubjectID("s1")
	newType := models.SubjectType("group")
	responder := updateSubjectAttempt(db, FAKE_ID, newId, newType)
	errorOut := responder.(*subjects.UpdateSubjectNotFound).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("subject, %s, not found", FAKE_ID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestUpdateSubjectDuplicate(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Insert the first subject into the database.
	s1Id := models.ExternalSubjectID("s1")
	s1Type := models.SubjectType("user")
	s1 := addSubject(db, s1Id, s1Type)

	// Insert the second subject into the database.
	s2Id := models.ExternalSubjectID("s2")
	s2Type := models.SubjectType("group")
	s2 := addSubject(db, s2Id, s2Type)

	// Attempt to change the ID of the second subject to be the same as the first.
	responder := updateSubjectAttempt(db, *s2.ID, *s1.SubjectID, *s2.SubjectType)
	errorOut := responder.(*subjects.UpdateSubjectBadRequest).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("another subject with the ID, %s, already exists", string(s1Id))
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestDeleteSubject(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Insert a subject into the database.
	s1Id := models.ExternalSubjectID("s1")
	s1Type := models.SubjectType("user")
	s1 := addSubject(db, s1Id, s1Type)

	// Delete the subject.
	deleteSubject(db, *s1.ID)

	// Verify that the subject was deleted.
	subjectList := listSubjects(db, nil, nil).Subjects
	if len(subjectList) != 0 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}
}

func TestDeleteSubjectNotFound(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Attempt to delete a non-existent subject.
	responder := deleteSubjectAttempt(db, models.InternalSubjectID(FAKE_ID))
	errorOut := responder.(*subjects.DeleteSubjectNotFound).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("subject, %s, not found", FAKE_ID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestDeleteSubjectByName(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Add a subject to the database.
	addSubject(db, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, models.ExternalSubjectID("b"), models.SubjectType("user"))

	// Delete the subject.
	deleteSubjectByExternalId(db, "a", "user")

	// Verify that the subject was deleted.
	subjectList := listSubjects(db, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}

	// Verify that the expected subject remains.
	checkSubject(t, subjectList, 0, "b", "user")
}

func TestDeleteSubjectByNameNotFound(t *testing.T) {
	if !shouldrun() {
		return
	}

	// Initialize the database.
	db := initdb(t)

	// Attempt to delete a subject.
	responder := deleteSubjectByExternalIdAttempt(db, "a", "user")
	errorOut := responder.(*subjects.DeleteSubjectByExternalIDNotFound).Payload

	// Verify that we got the expected error message.
	expected := "subject not found: user:a"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}
