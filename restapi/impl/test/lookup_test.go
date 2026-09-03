package test

import (
	"database/sql"
	"testing"

	"github.com/cyverse-de/permissions/models"
	impl "github.com/cyverse-de/permissions/restapi/impl/permissions"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"
	middleware "github.com/go-openapi/runtime/middleware"
)

func bySubjectAttempt(db *sql.DB, schema, subjectType, subjectID string, lookup bool, minLevel *string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildBySubjectHandler(db, schema)

	// Attempt to look up the permissions.
	params := permissions.BySubjectParams{
		HTTPRequest: fakeRequest(),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Lookup:      &lookup,
		MinLevel:    minLevel,
	}
	return handler(params)
}

func bySubject(db *sql.DB, schema, subjectType, subjectID string, lookup bool, minLevel *string) *models.PermissionList {
	responder := bySubjectAttempt(db, schema, subjectType, subjectID, lookup, minLevel)
	return responder.(*permissions.BySubjectOK).Payload
}

func bySubjectAndResourceTypeAttempt(
	db *sql.DB, schema, subjectType, subjectID, resourceType string, lookup bool, minLevel *string,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildBySubjectAndResourceTypeHandler(db, schema)

	// Attempt to look up the permissions.
	params := permissions.BySubjectAndResourceTypeParams{
		HTTPRequest:  fakeRequest(),
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		ResourceType: resourceType,
		Lookup:       &lookup,
		MinLevel:     minLevel,
	}
	return handler(params)
}

func bySubjectAndResourceType(
	db *sql.DB, schema, subjectType, subjectID, resourceType string, lookup bool, minLevel *string,
) *models.PermissionList {
	responder := bySubjectAndResourceTypeAttempt(db, schema, subjectType, subjectID, resourceType, lookup, minLevel)
	return responder.(*permissions.BySubjectAndResourceTypeOK).Payload
}

func bySubjectAndResourceAttempt(
	db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName string, lookup bool, minLevel *string,
) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildBySubjectAndResourceHandler(db, schema)

	// Attempt to look up the permissions.
	params := permissions.BySubjectAndResourceParams{
		HTTPRequest:  fakeRequest(),
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Lookup:       &lookup,
		MinLevel:     minLevel,
	}
	return handler(params)
}

func bySubjectAndResource(
	db *sql.DB, schema, subjectType, subjectID, resourceType, resourceName string, lookup bool, minLevel *string,
) *models.PermissionList {
	responder := bySubjectAndResourceAttempt(db, schema, subjectType, subjectID, resourceType, resourceName, lookup, minLevel)
	return responder.(*permissions.BySubjectAndResourceOK).Payload
}

func TestBySubject(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g2id", "analysis", "r3", "read")

	// Look up the permissions and verify that we get the expected number of results.
	perms := bySubject(db, schema, "user", "s2", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "r2", "g1id", "read")
}

func TestBySubjectMultiplePermissions(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "app", "r1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "write")

	// Look up the permissions and verify that we get the expected number of results.
	perms := bySubject(db, schema, "user", "s2", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "r2", "g1id", "write")
}

func TestBySubjectIncorrectSubjectType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "app", "r1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "write")

	// Attempt the lookup.
	responder := bySubjectAttempt(db, schema, "group", "s2", true, nil)
	errorOut := responder.(*permissions.BySubjectBadRequest).Payload
	expected := "incorrect type for subject, s2: group"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestBySubjectGroupsNotTransitive(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "app", "r1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "r3", "own")

	// Look up permissions and verify that we get the expected number of results.
	perms := bySubject(db, schema, "group", "g1id", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "r2", "g1id", "write")
}

func TestBySubjectNonLookup(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "app", "r1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "r3", "own")

	// List permissions for s2 and verify that we get the expected results.
	perms := bySubject(db, schema, "user", "s2", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "r2", "s2", "read")

	// List permissions for g1id and verify that we get the expected results.
	perms = bySubject(db, schema, "group", "g1id", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "r2", "g1id", "write")
}

func TestBySubjectMinLevel(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "r1", "own")
	putPermission(db, schema, "group", "g1id", "app", "r1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "r2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "r2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "r3", "own")
	putPermission(db, schema, "user", "s2", "app", "r4", "read")

	// The minimum level for this search.
	minLevel := "write"

	// List permissions for s2 and verify that we get the expected number of results.
	perms := bySubject(db, schema, "user", "s2", true, &minLevel).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "r1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "r2", "g1id", "write")
}

func TestBySubjectAndResourceType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")
	putPermission(db, schema, "user", "s2", "analysis", "analysis1", "own")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "analysis2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "analysis3", "own")

	// Look up app permissions and verify that we get the expected number of results.
	perms := bySubjectAndResourceType(db, schema, "user", "s2", "app", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "app2", "g1id", "write")

	// Look up analysis permissions and verify that we get the expected number of results.
	perms = bySubjectAndResourceType(db, schema, "user", "s2", "analysis", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "analysis1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "analysis2", "g1id", "write")
}

func TestBySubjectAndResourceTypeIncorrectSubjectType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// Look up permissions and verify that we get the expected number of results.
	responder := bySubjectAndResourceTypeAttempt(db, schema, "group", "s2", "app", true, nil)
	errorOut := responder.(*permissions.BySubjectAndResourceTypeBadRequest).Payload
	expected := "incorrect type for subject, s2: group"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestBySubjectAndResourceTypeUnknownResourceType(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// Look up permissions and verify that we get the expected number of results.
	perms := bySubjectAndResourceType(db, schema, "user", "s2", "blargle", true, nil).Permissions
	if len(perms) != 0 {
		t.Errorf("unexpected number of results: %d", len(perms))
	}
}

func TestBySubjectAndResourceTypeGroupsNotTransitive(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// Look up permissions and verify that we get the expected number of results.
	perms := bySubjectAndResourceType(db, schema, "group", "g1id", "app", true, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "app2", "g1id", "write")
}

func TestBySubjectAndResourceTypeNonLookup(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")
	putPermission(db, schema, "user", "s2", "analysis", "analysis1", "own")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "analysis2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "analysis3", "own")

	// Look up the app permissions for s2 and verify that we get the expected number of results.
	perms := bySubjectAndResourceType(db, schema, "user", "s2", "app", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "app2", "s2", "read")

	// Look up the app permissions for g1id and verify that we get the expected number of results.
	perms = bySubjectAndResourceType(db, schema, "group", "g1id", "app", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "app2", "g1id", "write")

	// Look up the analysis permissions for s2 and verify that we got the expected number of results.
	perms = bySubjectAndResourceType(db, schema, "user", "s2", "analysis", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "analysis1", "s2", "own")
	checkPermAtIndex(t, perms, 1, "analysis2", "s2", "read")

	// Look up the analysis permissions for g1id and verify that we get the expected number of results.
	perms = bySubjectAndResourceType(db, schema, "group", "g1id", "analysis", false, nil).Permissions
	if len(perms) != 2 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we go the expected results.
	checkPermAtIndex(t, perms, 0, "analysis1", "g1id", "read")
	checkPermAtIndex(t, perms, 1, "analysis2", "g1id", "write")
}

func TestBySubjectAndResourceTypeMinLevel(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")
	putPermission(db, schema, "user", "s2", "analysis", "analysis1", "own")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis1", "read")
	putPermission(db, schema, "user", "s2", "analysis", "analysis2", "read")
	putPermission(db, schema, "group", "g1id", "analysis", "analysis2", "write")
	putPermission(db, schema, "group", "g2id", "analysis", "analysis3", "own")

	// The minimum level for the search.
	minLevel := "own"

	// Look up app permissions and verify that we get the expected number of results.
	perms := bySubjectAndResourceType(db, schema, "user", "s2", "app", true, &minLevel).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")

	// Look up analysis permissions and verify that we get the expected number of results.
	perms = bySubjectAndResourceType(db, schema, "user", "s2", "analysis", true, &minLevel).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "analysis1", "s2", "own")
}

func TestBySubjectAndResource(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultGroupMemberships(t, db, schema)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// Look up permissions for app1 and verify that we get the expected number of results.
	perms := bySubjectAndResource(db, schema, "user", "s2", "app", "app1", true, nil).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")

	// Look up permissions for app2 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app2", true, nil).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app2", "g1id", "write")

	// Look up permissions for app3 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app3", true, nil).Permissions
	if len(perms) != 0 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}
}

func TestBySubjectAndResourceNonLookup(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// Look up permissions for app1 and verify that we get the expected number of results.
	perms := bySubjectAndResource(db, schema, "user", "s2", "app", "app1", false, nil).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")

	// Look up permissions for app2 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app2", false, nil).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app2", "s2", "read")

	// Look up permissions for app3 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app3", true, nil).Permissions
	if len(perms) != 0 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}
}

func TestBySubjectAndResourceMinLevel(t *testing.T) {
	skipUnlessIntegration(t)

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some permissions.
	putPermission(db, schema, "user", "s2", "app", "app1", "own")
	putPermission(db, schema, "group", "g1id", "app", "app1", "read")
	putPermission(db, schema, "user", "s2", "app", "app2", "read")
	putPermission(db, schema, "group", "g1id", "app", "app2", "write")
	putPermission(db, schema, "group", "g2id", "app", "app3", "own")

	// The  minimum level for the search.
	minLevel := "own"

	// Look up permissions for app1 and verify that we get the expected number of results.
	perms := bySubjectAndResource(db, schema, "user", "s2", "app", "app1", true, &minLevel).Permissions
	if len(perms) != 1 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Verify that we got the expected results.
	checkPermAtIndex(t, perms, 0, "app1", "s2", "own")

	// Look up permissions for app2 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app2", true, &minLevel).Permissions
	if len(perms) != 0 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}

	// Look up permissions for app3 and verify that we get the expected number of results.
	perms = bySubjectAndResource(db, schema, "user", "s2", "app", "app3", true, nil).Permissions
	if len(perms) != 0 {
		t.Fatalf("unexpected number of results: %d", len(perms))
	}
}
