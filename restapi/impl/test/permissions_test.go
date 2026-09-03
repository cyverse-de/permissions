package test

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"

	impl "github.com/cyverse-de/permissions/restapi/impl/permissions"
	middleware "github.com/go-openapi/runtime/middleware"
)

func fakeRequest() *http.Request {
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		logger.Log.Fatalf("unable to build a fake request: %v", err)
	}
	return request
}

func checkPermAtIndex(t *testing.T, ps []*models.Permission, i int32, resource, subject, level string) {
	p := ps[i]
	if *p.Resource.Name != resource {
		t.Errorf("unexpected resource in result %d: %s", i, *p.Resource.Name)
	}
	if string(*p.Subject.SubjectID) != subject {
		t.Errorf("unexpected subject in result %d: %s", i, string(*p.Subject.SubjectID))
	}
	if string(*p.PermissionLevel) != level {
		t.Errorf("unexpected permission level in result %d: %s", i, string(*p.PermissionLevel))
	}
}

func findPerm(perms []*models.Permission, subjectID string) *models.Permission {
	sid := models.ExternalSubjectID(subjectID)
	for _, perm := range perms {
		if *perm.Subject.SubjectID == sid {
			return perm
		}
	}
	return nil
}

func checkPermForSubject(t *testing.T, ps []*models.Permission, resource, subject, level string) {
	p := findPerm(ps, subject)
	if p == nil {
		t.Fatalf("no permission found for %s", subject)
	}
	if *p.Resource.Name != resource {
		t.Errorf("unexpected resource in permission for %s: %s", subject, *p.Resource.Name)
	}
	if string(*p.PermissionLevel) != level {
		t.Errorf("unexpected permission level in permission for %s: %s", subject, string(*p.PermissionLevel))
	}
}

func grantPermissionAttempt(
	db *sql.DB,
	schema string,
	subject *models.SubjectIn,
	resource *models.ResourceIn,
	level models.PermissionLevel,
) middleware.Responder {

	handler := impl.BuildGrantPermissionHandler(db, schema, testUserSuffix)

	// Attempt to add the permission.
	req := &models.PermissionGrantRequest{Subject: subject, Resource: resource, PermissionLevel: &level}
	params := permissions.GrantPermissionParams{
		HTTPRequest:            fakeRequest(),
		PermissionGrantRequest: req,
	}
	return handler(params)
}

func grantPermission(
	db *sql.DB,
	schema string,
	subject *models.SubjectIn,
	resource *models.ResourceIn,
	level models.PermissionLevel,
) *models.Permission {
	responder := grantPermissionAttempt(db, schema, subject, resource, level)
	return responder.(*permissions.GrantPermissionOK).Payload
}

func revokePermissionAttempt(
	db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName string,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildRevokePermissionHandler(db, schema)

	// Attempt to revoke the permission.
	params := permissions.RevokePermissionParams{
		HTTPRequest:  fakeRequest(),
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		ResourceType: resourceType,
		ResourceName: resourceName,
	}
	return handler(params)
}

func revokePermission(db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName string) {
	responder := revokePermissionAttempt(db, schema, subjectType, subjectID, resourceType, resourceName)
	_ = responder.(*permissions.RevokePermissionOK)
}

func putPermissionAttempt(
	db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName, level string,
) middleware.Responder {

	handler := impl.BuildPutPermissionHandler(db, schema, testUserSuffix)

	// Attempt to put the permission.
	permissionLevel := models.PermissionLevel(level)
	params := permissions.PutPermissionParams{
		HTTPRequest:  fakeRequest(),
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Permission:   &models.PermissionPutRequest{PermissionLevel: &permissionLevel},
	}
	return handler(params)
}

func putPermission(db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName, level string) *models.Permission {
	responder := putPermissionAttempt(db, schema, subjectType, subjectID, resourceType, resourceName, level)
	return responder.(*permissions.PutPermissionOK).Payload
}

func listPermissionsAttempt(db *sql.DB, schema string) middleware.Responder {

	handler := impl.BuildListPermissionsHandler(db, schema)

	// Attempt to list the permissions.
	return handler(permissions.ListPermissionsParams{HTTPRequest: fakeRequest()})
}

func listPermissions(db *sql.DB, schema string) *models.PermissionList {
	responder := listPermissionsAttempt(db, schema)
	return responder.(*permissions.ListPermissionsOK).Payload
}

func listResourcePermissionsAttempt(
	db *sql.DB,
	schema, resourceType, resourceName string,
	expandGroups bool,
	maxExpandedSubjects int,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildListResourcePermissionsHandler(db, schema, maxExpandedSubjects)

	// Attempt to list the permissions for the resource.
	params := permissions.ListResourcePermissionsParams{
		HTTPRequest:  fakeRequest(),
		ResourceType: resourceType,
		ResourceName: resourceName,
		ExpandGroups: &expandGroups,
	}
	return handler(params)
}

func listResourcePermissions(
	db *sql.DB,
	schema, resourceType, resourceName string,
	expandGroups bool,
) *models.PermissionList {
	responder := listResourcePermissionsAttempt(db, schema, resourceType, resourceName, expandGroups, testMaxExpandedSubjects)
	return responder.(*permissions.ListResourcePermissionsOK).Payload
}

func listSubjectPermissionsAttempt(db *sql.DB, schema, subjectType, subjectID string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildBySubjectHandler(db, schema)

	// Attempt to look up the permissions.
	lookup := false
	params := permissions.BySubjectParams{
		HTTPRequest: fakeRequest(),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Lookup:      &lookup,
		MinLevel:    nil,
	}
	return handler(params)
}

func listSubjectPermissions(db *sql.DB, schema, subjectType, subjectID string) *models.PermissionList {
	responder := listSubjectPermissionsAttempt(db, schema, subjectType, subjectID)
	return responder.(*permissions.BySubjectOK).Payload
}

func copyPermissionsAttempt(db *sql.DB, schema, sourceType, sourceID, destType, destID string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildCopyPermissionsHandler(db, schema, testUserSuffix)

	// Attempt to copy the permissions.
	destinationSubjectType := models.SubjectType(destType)
	destinationSubjectID := models.ExternalSubjectID(destID)
	dest := models.SubjectIn{
		SubjectType: &destinationSubjectType,
		SubjectID:   &destinationSubjectID,
	}
	params := permissions.CopyPermissionsParams{
		HTTPRequest:  fakeRequest(),
		SubjectType:  sourceType,
		SubjectID:    sourceID,
		DestSubjects: &models.SubjectsIn{Subjects: []*models.SubjectIn{&dest}},
	}
	return handler(params)
}

func copyPermissions(db *sql.DB, schema, sourceType, sourceID, destType, destID string) middleware.Responder {
	responder := copyPermissionsAttempt(db, schema, sourceType, sourceID, destType, destID)
	return responder.(*permissions.CopyPermissionsOK)
}

func addDefaultPermissions(db *sql.DB, schema string) {
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "group", "g2id", "app", "app1", "write")
	putPermission(db, schema, "user", "s3", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")
	putPermission(db, schema, "user", "s2", "analysis", "analysis1", "own")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis1", "read")
	putPermission(db, schema, "group", "g2id", "analysis", "analysis1", "write")
	putPermission(db, schema, "user", "s3", "analysis", "analysis1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "analysis2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "analysis3", "own")
}

func TestGrantPermission(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	permission := grantPermission(db, schema, subjectIn, resourceIn, "own")

	// Verify that we got the expected result.
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID returned: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID returned: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type returned: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID returned: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name returned: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type returned: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level returned: %v", permission.PermissionLevel)
	}
}

func TestListPermissions(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	_ = grantPermission(db, schema, subjectIn, resourceIn, "own")

	// List the permissions and verify that we get the expected number of results.
	permissions := listPermissions(db, schema).Permissions
	if len(permissions) != 1 {
		t.Fatalf("unexpected number of permissions listed: %d", len(permissions))
	}

	// Verify that we got the expected result.
	permission := permissions[0]
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID listed: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID listed: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type listed: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID listed: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name listed: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type listed: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level listed: %v", permission.PermissionLevel)
	}
}

func TestAutoInsertSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Create, but don't register, a subject.
	subjectIn := newSubjectIn("s1", "user")

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	_ = grantPermission(db, schema, subjectIn, resourceIn, "own")

	// List the permissions and verify that we get the expected number of results.
	permissions := listPermissions(db, schema).Permissions
	if len(permissions) != 1 {
		t.Fatalf("unexpected number of permissions listed: %d", len(permissions))
	}

	// Verify that we got the expected result.
	permission := permissions[0]
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if len(*permission.Subject.ID) != 36 {
		t.Errorf("unexpected internal subject ID listed: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectIn.SubjectID {
		t.Errorf("unexpected external subject ID listed: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectIn.SubjectType {
		t.Errorf("unexpedted subject type listed: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID listed: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name listed: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type listed: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level listed: %v", permission.PermissionLevel)
	}
}

func TestAutoInsertResource(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Create, but don't register, a subject.
	resourceIn := newResourceIn("r1", "app")

	// Grant the subject access to the resource.
	_ = grantPermission(db, schema, subjectIn, resourceIn, "own")

	// List the permissions and verify that we get the expected number of results.
	permissions := listPermissions(db, schema).Permissions
	if len(permissions) != 1 {
		t.Fatalf("unexpected number of permissions listed: %d", len(permissions))
	}

	// Verify that we got the expected result.
	permission := permissions[0]
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID listed: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID listed: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type listed: %s", *permission.Subject.SubjectType)
	}
	if permission.Resource.ID == nil {
		t.Error("no resource ID listed")
	}
	if *permission.Resource.Name != *resourceIn.Name {
		t.Errorf("unexpected resource name listed: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceIn.ResourceType {
		t.Errorf("unexpected resource type listed: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level listed: %v", permission.PermissionLevel)
	}
}

func TestUpdatePermissionLevel(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	_ = grantPermission(db, schema, subjectIn, resourceIn, "own")

	// Revise the ownership level.
	_ = grantPermission(db, schema, subjectIn, resourceIn, "write")

	// List the permissions and verify that we get the expected number of results.
	permissions := listPermissions(db, schema).Permissions
	if len(permissions) != 1 {
		t.Fatalf("unexpected number of permissions listed: %d", len(permissions))
	}

	// Verify that we got the expected result.
	permission := permissions[0]
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID listed: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID listed: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type listed: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID listed: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name listed: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type listed: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("write") {
		t.Errorf("unexpected permission level listed: %v", permission.PermissionLevel)
	}
}

func TestRevokePermission(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the datasbase.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Grant the subject access to the resource.
	_ = grantPermission(db, schema, newSubjectIn("s1", "user"), newResourceIn("r1", "app"), "own")

	// Revoke the permission.
	revokePermission(db, schema, "user", "s1", "app", "r1")

	// List the permissions and verify that we get the expected number of results.
	permissions := listPermissions(db, schema).Permissions
	if len(permissions) != 0 {
		t.Errorf("unexpected number of permissions listed: %d", len(permissions))
	}
}

func TestRevokePermissionResourceTypeNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to revoke a permission for a bogus resource type.
	responder := revokePermissionAttempt(db, schema, "user", "s1", "foo", "r1")
	errorOut := responder.(*permissions.RevokePermissionNotFound).Payload

	// Verify that we got the expected error message.
	expected := "resource type not found: foo"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestRevokePermissionResourceNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to revoke a permission for a bogus resource.
	responder := revokePermissionAttempt(db, schema, "user", "s1", "app", "r1")
	errorOut := responder.(*permissions.RevokePermissionNotFound).Payload

	// Verify that we got the expected error message.
	expected := "resource not found: app/r1"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestRevokePermissionUserNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	_ = addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Attempt to revoke a permission for a bogus user.
	responder := revokePermissionAttempt(db, schema, "user", "nobody", "app", "r1")
	errorOut := responder.(*permissions.RevokePermissionNotFound).Payload

	// Verify that we got the expected error message.
	expected := "subject not found: user/nobody"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestRevokePermissionNotFound(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	_ = addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	_ = addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Attempt to revoke a permission for a bogus user.
	responder := revokePermissionAttempt(db, schema, "user", "s1", "app", "r1")
	errorOut := responder.(*permissions.RevokePermissionNotFound).Payload

	// Verify that we got the expected error message.
	expected := "permission not found: app/r1:user/s1"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestPutPermission(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	permission := putPermission(db, schema, "user", "s1", "app", "r1", "own")

	// Verify that we got the expected result.
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID returned: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID returned: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type returned: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID returned: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name returned: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type returned: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level returned: %v", permission.PermissionLevel)
	}
}

func TestPutPermissionNewSubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define, but don't register a subject.
	subjectIn := newSubjectIn("s1", "user")

	// Define a resource.
	resourceIn := newResourceIn("r1", "app")
	resourceOut := addResource(db, schema, *resourceIn.Name, *resourceIn.ResourceType)

	// Grant the subject access to the resource.
	permission := putPermission(db, schema, "user", "s1", "app", "r1", "own")

	// Verify that we got the expected result.
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if len(*permission.Subject.ID) != 36 {
		t.Errorf("unexpected internal subject ID returned: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectIn.SubjectID {
		t.Errorf("unexpected external subject ID returned: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectIn.SubjectType {
		t.Errorf("unexpedted subject type returned: %s", *permission.Subject.SubjectType)
	}
	if *permission.Resource.ID != *resourceOut.ID {
		t.Errorf("unexpected resource ID returned: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceOut.Name {
		t.Errorf("unexpected resource name returned: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceOut.ResourceType {
		t.Errorf("unexpected resource type returned: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level returned: %v", permission.PermissionLevel)
	}
}

func TestPutPermissionNewResource(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	subjectOut := addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Define, but don't register a resource.
	resourceIn := newResourceIn("r1", "app")

	// Grant the subject access to the resource.
	permission := putPermission(db, schema, "user", "s1", "app", "r1", "own")

	// Verify that we got the expected result.
	if len(*permission.ID) != 36 {
		t.Errorf("unexpected internal permission ID returned: %s", *permission.ID)
	}
	if *permission.Subject.ID != *subjectOut.ID {
		t.Errorf("unexpected internal subject ID returned: %s", *permission.Subject.ID)
	}
	if *permission.Subject.SubjectID != *subjectOut.SubjectID {
		t.Errorf("unexpected external subject ID returned: %s", *permission.Subject.SubjectID)
	}
	if *permission.Subject.SubjectType != *subjectOut.SubjectType {
		t.Errorf("unexpedted subject type returned: %s", *permission.Subject.SubjectType)
	}
	if len(*permission.Resource.ID) != 36 {
		t.Errorf("unexpected resource ID returned: %s", *permission.Resource.ID)
	}
	if *permission.Resource.Name != *resourceIn.Name {
		t.Errorf("unexpected resource name returned: %s", *permission.Resource.Name)
	}
	if *permission.Resource.ResourceType != *resourceIn.ResourceType {
		t.Errorf("unexpected resource type returned: %s", *permission.Resource.ResourceType)
	}
	if *permission.PermissionLevel != models.PermissionLevel("own") {
		t.Errorf("unexpected permission level returned: %v", permission.PermissionLevel)
	}
}

func TestPutPermissionDuplicateSubjectId(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Define a subject.
	subjectIn := newSubjectIn("s1", "user")
	_ = addSubject(db, schema, *subjectIn.SubjectID, *subjectIn.SubjectType)

	// Attempt to add a permission using a duplicate subject ID.
	responder := putPermissionAttempt(db, schema, "group", "s1", "app", "r1", "own")
	errorOut := responder.(*permissions.PutPermissionBadRequest).Payload

	// Verify that we got the expected error.
	expected := "another subject with ID, s1, already exists"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestPutPermissionBogusResourceType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to add a permission using a bogus resource type.
	responder := putPermissionAttempt(db, schema, "user", "s1", "foo", "r1", "own")
	errorOut := responder.(*permissions.PutPermissionBadRequest).Payload

	// Verify that we got the expected error.
	expected := "no resource type named, foo, found"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestListResourcePermissionsEmpty(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// List permissions and verify that we get the expected number of results.
	perms := listResourcePermissions(db, schema, "app", "r1", false).Permissions
	if len(perms) != 0 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}
}

func TestListResourcePermissions(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// List permissions and verify that we get the expected number of results.
	perms := listResourcePermissions(db, schema, "app", "app1", false).Permissions
	if len(perms) != 4 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "app1", "g2id", "write")
	checkPermAtIndex(t, perms, 2, "app1", "s2", "own")
	checkPermAtIndex(t, perms, 3, "app1", "s3", "read")
}

func TestListResourcePermissionsExpandGroups(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// List permissions and verify that we get the expected number of results.
	perms := listResourcePermissions(db, schema, "app", "app1", true).Permissions
	if len(perms) != 3 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermForSubject(t, perms, "app1", "s1", "write")
	checkPermForSubject(t, perms, "app1", "s2", "own")
	checkPermForSubject(t, perms, "app1", "s3", "read")

	// List permissions for a second resource and verify that we get the expected number of results.
	perms = listResourcePermissions(db, schema, "app", "app2", true).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermForSubject(t, perms, "app2", "s1", "write")
	checkPermForSubject(t, perms, "app2", "s2", "write")
}

func TestListResourcePermissionsExpandGroupsTooManySubjects(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// The pre-check counts app1 as four subjects (two direct user grants plus
	// two distinct group members), so a cap of two must refuse the expansion.
	responder := listResourcePermissionsAttempt(db, schema, "app", "app1", true, 2)
	errorOut := responder.(*permissions.ListResourcePermissionsInternalServerError).Payload
	if !strings.Contains(*errorOut.Reason, "too many subjects") {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}

	// A cap that accommodates the pre-check count must still serve the request.
	responder = listResourcePermissionsAttempt(db, schema, "app", "app1", true, 4)
	perms := responder.(*permissions.ListResourcePermissionsOK).Payload.Permissions
	if len(perms) != 3 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}
}

func TestCopyPermissions(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// Copy permissions from subject s2 to subject s1.
	copyPermissions(db, schema, "user", "s2", "user", "s1")

	// Verify that the permissions were copied.
	perms := listSubjectPermissions(db, schema, "user", "s1").Permissions
	if len(perms) != 4 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s1", "own")
	checkPermAtIndex(t, perms, 1, "app2", "s1", "read")
	checkPermAtIndex(t, perms, 2, "analysis1", "s1", "own")
	checkPermAtIndex(t, perms, 3, "analysis2", "s1", "read")
}

func TestCopyPermissionsOverwrite(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// Add a permission that should be overwritten.
	putPermission(db, schema, "user", "s1", "app", "app1", "read")

	// Copy permissions from subject s2 to subject s1.
	copyPermissions(db, schema, "user", "s2", "user", "s1")

	// Verify that the permissions were copied.
	perms := listSubjectPermissions(db, schema, "user", "s1").Permissions
	if len(perms) != 4 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s1", "own")
	checkPermAtIndex(t, perms, 1, "app2", "s1", "read")
	checkPermAtIndex(t, perms, 2, "analysis1", "s1", "own")
	checkPermAtIndex(t, perms, 3, "analysis2", "s1", "read")
}

func TestCopyPermissionsRetain(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	addDefaultPermissions(db, schema)

	// Add a permission that should not be overwritten
	putPermission(db, schema, "user", "s1", "app", "app2", "own")

	// Copy permissions from subject s2 to subject s1.
	copyPermissions(db, schema, "user", "s2", "user", "s1")

	// Verify that the permissions were copied.
	perms := listSubjectPermissions(db, schema, "user", "s1").Permissions
	if len(perms) != 4 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s1", "own")
	checkPermAtIndex(t, perms, 1, "app2", "s1", "own")
	checkPermAtIndex(t, perms, 2, "analysis1", "s1", "own")
	checkPermAtIndex(t, perms, 3, "analysis2", "s1", "read")
}
