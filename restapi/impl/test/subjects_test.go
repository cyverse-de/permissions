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

func checkSubject(t *testing.T, subjects []*models.SubjectOut, i int32, subjectID, subjectType string) {
	actual := subjects[i]
	if *actual.SubjectID != models.ExternalSubjectID(subjectID) {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if *actual.SubjectType != models.SubjectType(subjectType) {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func addSubjectAttempt(
	db *sql.DB,
	schema string,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildAddSubjectHandler(db, schema, testUserSuffix)

	// Attempt to add the subject to the database.
	subjectIn := &models.SubjectIn{SubjectID: &subjectID, SubjectType: &subjectType}
	params := subjects.AddSubjectParams{
		HTTPRequest: fakeRequest(),
		SubjectIn:   subjectIn,
	}
	return handler(params)
}

func addSubject(db *sql.DB, schema string, subjectID models.ExternalSubjectID, subjectType models.SubjectType) *models.SubjectOut {
	responder := addSubjectAttempt(db, schema, subjectID, subjectType)
	return responder.(*subjects.AddSubjectCreated).Payload
}

func listSubjectsAttempt(db *sql.DB, schema string, subjectType, subjectID *string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildListSubjectsHandler(db, schema)

	// Attempt to list the subjects.
	params := subjects.ListSubjectsParams{
		HTTPRequest: fakeRequest(),
		SubjectType: subjectType,
		SubjectID:   subjectID,
	}
	return handler(params)
}

func listSubjects(db *sql.DB, schema string, subjectType, subjectID *string) *models.SubjectsOut {
	responder := listSubjectsAttempt(db, schema, subjectType, subjectID)
	return responder.(*subjects.ListSubjectsOK).Payload
}

func updateSubjectAttempt(
	db *sql.DB,
	schema string,
	id models.InternalSubjectID,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildUpdateSubjectHandler(db, schema, testUserSuffix)

	// Attempt to update the subject.
	subjectIn := &models.SubjectIn{SubjectID: &subjectID, SubjectType: &subjectType}
	params := subjects.UpdateSubjectParams{
		HTTPRequest: fakeRequest(),
		ID:          string(id),
		SubjectIn:   subjectIn,
	}
	return handler(params)
}

func updateSubject(
	db *sql.DB,
	schema string,
	id models.InternalSubjectID,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) *models.SubjectOut {
	responder := updateSubjectAttempt(db, schema, id, subjectID, subjectType)
	return responder.(*subjects.UpdateSubjectOK).Payload
}

func deleteSubjectAttempt(db *sql.DB, schema string, id models.InternalSubjectID) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteSubjectHandler(db, schema)

	// Attempt to delete the subject.
	params := subjects.DeleteSubjectParams{
		HTTPRequest: fakeRequest(),
		ID:          string(id),
	}
	return handler(params)
}

func deleteSubject(db *sql.DB, schema string, id models.InternalSubjectID) {
	responder := deleteSubjectAttempt(db, schema, id)
	_ = responder.(*subjects.DeleteSubjectOK)
}

func deleteSubjectByExternalIDAttempt(db *sql.DB, schema, subjectID, subjectType string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteSubjectByExternalIDHandler(db, schema)

	// Attempt to delete the subject.
	params := subjects.DeleteSubjectByExternalIDParams{
		HTTPRequest: fakeRequest(),
		SubjectID:   subjectID,
		SubjectType: subjectType,
	}
	return handler(params)
}

func deleteSubjectByExternalID(db *sql.DB, schema, subjectID, subjectType string) {
	responder := deleteSubjectByExternalIDAttempt(db, schema, subjectID, subjectType)
	_ = responder.(*subjects.DeleteSubjectByExternalIDOK)
}

func TestAddSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	subjectID := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	subject := addSubject(db, schema, subjectID, subjectType)

	// Verify that we got the expected response.
	if *subject.SubjectID != subjectID {
		t.Errorf("unexpected subject ID: %s", *subject.SubjectID)
	}
	if *subject.SubjectType != subjectType {
		t.Errorf("unexpected subject type: %s", *subject.SubjectType)
	}
}

func TestAddDuplicateSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	subjectID := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	addSubject(db, schema, subjectID, subjectType)

	// Attempt to add a subject with the same ID.
	responder := addSubjectAttempt(db, schema, subjectID, subjectType)
	errorOut := responder.(*subjects.AddSubjectBadRequest).Payload

	// Verify that we got the expected result.
	expected := fmt.Sprintf("subject, %s, already exists", string(subjectID))
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestListSubjects(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	subjectID := models.ExternalSubjectID("nobody")
	subjectType := models.SubjectType("user")
	expected := addSubject(db, schema, subjectID, subjectType)

	// List the subjects and verify that we get the expected number of results.
	subjectList := listSubjects(db, schema, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	actual := subjectList[0]
	if *expected.ID != *actual.ID {
		t.Errorf("unexpected ID: %s", string(*actual.ID))
	}
	if *expected.SubjectID != *actual.SubjectID {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if *expected.SubjectType != *actual.SubjectType {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func TestListSubjectsByExternalId(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	expected := addSubject(db, schema, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, schema, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectID := "a"
	subjectList := listSubjects(db, schema, nil, &subjectID).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	actual := subjectList[0]
	if *expected.ID != *actual.ID {
		t.Errorf("unexpected ID: %s", string(*actual.ID))
	}
	if *expected.SubjectID != *actual.SubjectID {
		t.Errorf("unexpected subject ID: %s", string(*actual.SubjectID))
	}
	if *expected.SubjectType != *actual.SubjectType {
		t.Errorf("unexpected subject type: %s", string(*actual.SubjectType))
	}
}

func TestListSubjectsByType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	addSubject(db, schema, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, schema, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectType := "user"
	subjectList := listSubjects(db, schema, &subjectType, nil).Subjects
	if len(subjectList) != 2 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	checkSubject(t, subjectList, 0, "a", "user")
	checkSubject(t, subjectList, 1, "c", "user")
}

func TestListSubjectsByExternalIdAndType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject.
	addSubject(db, schema, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("b"), models.SubjectType("group"))
	addSubject(db, schema, models.ExternalSubjectID("c"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("d"), models.SubjectType("group"))

	// List the subjects and verify that we get the expected number of results.
	subjectID := "a"
	subjectType := "user"
	subjectList := listSubjects(db, schema, &subjectType, &subjectID).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of subjects listed: %d", len(subjectList))
	}

	// Verify that we got the expected result.
	checkSubject(t, subjectList, 0, "a", "user")
}

func TestListSubjectsEmpty(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// List the subjects and verify that we get the expected result.
	subjectList := listSubjects(db, schema, nil, nil).Subjects
	if subjectList == nil {
		t.Error("nil value returned as a subject list")
	}
	if len(subjectList) != 0 {
		t.Errorf("unexpected number of results: %d", len(subjectList))
	}
}

func TestUpdateSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject to the database.
	origID := models.ExternalSubjectID("s1")
	origType := models.SubjectType("user")
	orig := addSubject(db, schema, origID, origType)

	// Change the subject ID and type.
	newID := models.ExternalSubjectID("s2")
	newType := models.SubjectType("group")
	new := updateSubject(db, schema, *orig.ID, newID, newType)

	// Verify that we got the expected result.
	if *new.ID != *orig.ID {
		t.Errorf("unexpected internal ID returned: %s", *new.ID)
	}
	if *new.SubjectID != newID {
		t.Errorf("unexpected external ID returned: %s", *new.SubjectID)
	}
	if *new.SubjectType != newType {
		t.Errorf("unexpected subject type returned: %s", *new.SubjectType)
	}

	// List the subjects and verify that we get the expected number of results.
	subjectList := listSubjects(db, schema, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}

	// Verify that we get the expected result.
	listed := subjectList[0]
	if *listed.ID != *orig.ID {
		t.Errorf("unexpected internal ID listed: %s", *listed.ID)
	}
	if *listed.SubjectID != newID {
		t.Errorf("unexpected external ID listed: %s", *listed.SubjectID)
	}
	if *listed.SubjectType != newType {
		t.Errorf("unexpected subject type listed: %s", *listed.SubjectType)
	}
}

func TestUpdateSubjectNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Attempt to update a subject.
	newID := models.ExternalSubjectID("s1")
	newType := models.SubjectType("group")
	responder := updateSubjectAttempt(db, schema, FakeID, newID, newType)
	errorOut := responder.(*subjects.UpdateSubjectNotFound).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("subject, %s, not found", FakeID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestUpdateSubjectDuplicate(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Insert the first subject into the database.
	s1Id := models.ExternalSubjectID("s1")
	s1Type := models.SubjectType("user")
	s1 := addSubject(db, schema, s1Id, s1Type)

	// Insert the second subject into the database.
	s2Id := models.ExternalSubjectID("s2")
	s2Type := models.SubjectType("group")
	s2 := addSubject(db, schema, s2Id, s2Type)

	// Attempt to change the ID of the second subject to be the same as the first.
	responder := updateSubjectAttempt(db, schema, *s2.ID, *s1.SubjectID, *s2.SubjectType)
	errorOut := responder.(*subjects.UpdateSubjectBadRequest).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("another subject with the ID, %s, already exists", string(s1Id))
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestDeleteSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Insert a subject into the database.
	s1Id := models.ExternalSubjectID("s1")
	s1Type := models.SubjectType("user")
	s1 := addSubject(db, schema, s1Id, s1Type)

	// Delete the subject.
	deleteSubject(db, schema, *s1.ID)

	// Verify that the subject was deleted.
	subjectList := listSubjects(db, schema, nil, nil).Subjects
	if len(subjectList) != 0 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}
}

func TestDeleteSubjectNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Attempt to delete a non-existent subject.
	responder := deleteSubjectAttempt(db, schema, models.InternalSubjectID(FakeID))
	errorOut := responder.(*subjects.DeleteSubjectNotFound).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("subject, %s, not found", FakeID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestDeleteSubjectByName(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Add a subject to the database.
	addSubject(db, schema, models.ExternalSubjectID("a"), models.SubjectType("user"))
	addSubject(db, schema, models.ExternalSubjectID("b"), models.SubjectType("user"))

	// Delete the subject.
	deleteSubjectByExternalID(db, schema, "a", "user")

	// Verify that the subject was deleted.
	subjectList := listSubjects(db, schema, nil, nil).Subjects
	if len(subjectList) != 1 {
		t.Fatalf("unexpected number of results: %d", len(subjectList))
	}

	// Verify that the expected subject remains.
	checkSubject(t, subjectList, 0, "b", "user")
}

func TestDeleteSubjectByNameNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)

	// Attempt to delete a subject.
	responder := deleteSubjectByExternalIDAttempt(db, schema, "a", "user")
	errorOut := responder.(*subjects.DeleteSubjectByExternalIDNotFound).Payload

	// Verify that we got the expected error message.
	expected := "subject not found: user:a"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}
