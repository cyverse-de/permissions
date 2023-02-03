// nolint
package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/cyverse-de/permissions/models"
	"github.com/cyverse-de/permissions/restapi/operations/resources"

	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
	impl "github.com/cyverse-de/permissions/restapi/impl/resources"
	middleware "github.com/go-openapi/runtime/middleware"
)

func listResourcesDirectly(db *sql.DB, schema string, t *testing.T) []*models.ResourceOut {

	// Start a transaction.
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("unable to list resources: %s", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
	if err != nil {
		t.Fatalf("unable to list resources: %s", err)
	}

	// List the resources.
	resources, err := permsdb.ListResources(context.Background(), tx, nil, nil)
	if err != nil {
		t.Fatalf("unable to list resources: %s", err)
	}

	return resources
}

func addResourceAttempt(db *sql.DB, schema, name, resourceType string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildAddResourceHandler(db, schema)

	// Attempt to add the resource to the database.
	resourceIn := &models.ResourceIn{Name: &name, ResourceType: &resourceType}
	params := resources.AddResourceParams{ResourceIn: resourceIn}
	return handler(params)
}

func addResource(db *sql.DB, schema, name, resourceType string) *models.ResourceOut {
	responder := addResourceAttempt(db, schema, name, resourceType)
	return responder.(*resources.AddResourceCreated).Payload
}

func listResourcesAttempt(db *sql.DB, schema string, resourceType, name *string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildListResourcesHandler(db, schema)

	// Attempt to list the resources.
	params := resources.ListResourcesParams{ResourceTypeName: resourceType, ResourceName: name}
	return handler(params)
}

func listResources(db *sql.DB, schema string, resourceType, name *string) *models.ResourcesOut {
	responder := listResourcesAttempt(db, schema, resourceType, name)
	return responder.(*resources.ListResourcesOK).Payload
}

func updateResourceAttempt(db *sql.DB, schema, id, name string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildUpdateResourceHandler(db, schema)

	// Attempt to update the resource.
	resourceUpdate := &models.ResourceUpdate{Name: &name}
	params := resources.UpdateResourceParams{ID: id, ResourceUpdate: resourceUpdate}
	return handler(params)
}

func updateResource(db *sql.DB, schema, id, name string) *models.ResourceOut {
	responder := updateResourceAttempt(db, schema, id, name)
	return responder.(*resources.UpdateResourceOK).Payload
}

func deleteResourceAttempt(db *sql.DB, schema, id string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteResourceHandler(db, schema)

	// Attempt to delete the resource.
	params := resources.DeleteResourceParams{ID: id}
	return handler(params)
}

func deleteResource(db *sql.DB, schema, id string) {
	responder := deleteResourceAttempt(db, schema, id)
	_ = responder.(*resources.DeleteResourceOK)
}

func deleteResourceByNameAttempt(db *sql.DB, schema, resourceTypeName, name string) middleware.Responder {

	// Build the request handler.
	handler := impl.BuildDeleteResourceByNameHandler(db, schema)

	// Attempt to delete the resource.
	params := resources.DeleteResourceByNameParams{ResourceTypeName: resourceTypeName, ResourceName: name}
	return handler(params)
}

func deleteResourceByName(db *sql.DB, schema, resourceTypeName, name string) {
	responder := deleteResourceByNameAttempt(db, schema, resourceTypeName, name)
	_ = responder.(*resources.DeleteResourceByNameOK)
}

func TestAddResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource.
	resourceName := "resource"
	resourceType := "app"
	resource := addResource(db, schema, resourceName, resourceType)

	// Verify the name and description.
	if *resource.Name != resourceName {
		t.Errorf("unexpected resource name: %s", *resource.Name)
	}
	if *resource.ResourceType != resourceType {
		t.Errorf("unexpected resource type: %s", *resource.ResourceType)
	}

	// List the resources and verify that we got the expected number of results.
	resources := listResourcesDirectly(db, schema, t)
	if len(resources) != 1 {
		t.Fatalf("unexpected number of resource types: %d", len(resources))
	}

	// Verify that we got the expected result.
	resource = resources[0]
	if *resource.Name != resourceName {
		t.Errorf("unexpected resource name listed: %s", *resource.Name)
	}
	if *resource.ResourceType != resourceType {
		t.Errorf("unexpected resource type listed: %s", *resource.ResourceType)
	}
}

func TestAddDuplicateResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addResource(db, schema, "resource", "app")

	// Attempt to add a duplicate resource.
	resourceName := "resource"
	resourceType := "app"
	responder := addResourceAttempt(db, schema, resourceName, resourceType)
	errorOut := responder.(*resources.AddResourceBadRequest).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("a resource named, '%s', with type, '%s', already exists", resourceName, resourceType)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestAddResourceInvalidType(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to add a resource with an invalid type.
	resourceName := "resource"
	resourceType := "invisible_resource_type"
	responder := addResourceAttempt(db, schema, resourceName, resourceType)
	errorOut := responder.(*resources.AddResourceBadRequest).Payload

	// Verify that we got the expected error message.
	expected := fmt.Sprintf("no resource type named, '%s', found", resourceType)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure reason: %s", *errorOut.Reason)
	}
}

func TestListResources(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource to the database.
	r1 := addResource(db, schema, "r1", "app")

	// List the resources and verify we get the expected number of results.
	result := listResources(db, schema, nil, nil)
	if len(result.Resources) != 1 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}

	// Verify that we got the expected result.
	resource := result.Resources[0]
	if *resource.Name != *r1.Name {
		t.Errorf("unexpected resource name: %s", *resource.Name)
	}
	if *resource.ResourceType != *r1.ResourceType {
		t.Errorf("unexpected resource type: %s", *resource.ResourceType)
	}
}

func TestListResourcesByName(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some resources to the database.
	r1 := addResource(db, schema, "r1", "app")
	addResource(db, schema, "r2", "app")

	// List the resources and verify we get the expected number of results.
	result := listResources(db, schema, nil, r1.Name)
	if len(result.Resources) != 1 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}

	// Verify that we got the expected result.
	resource := result.Resources[0]
	if *resource.Name != *r1.Name {
		t.Errorf("unexpected resource name: %s", *resource.Name)
	}
	if *resource.ResourceType != *r1.ResourceType {
		t.Errorf("unexpected resource type: %s", *resource.ResourceType)
	}
}

func TestListResourcesByType(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some resources to the database.
	r1 := addResource(db, schema, "r1", "app")
	addResource(db, schema, "r1", "analysis")

	// List the resources and verify we get the expected number of results.
	result := listResources(db, schema, r1.ResourceType, nil)
	if len(result.Resources) != 1 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}

	// Verify that we got the expected result.
	resource := result.Resources[0]
	if *resource.Name != *r1.Name {
		t.Errorf("unexpected resource name: %s", *resource.Name)
	}
	if *resource.ResourceType != *r1.ResourceType {
		t.Errorf("unexpected resource type: %s", *resource.ResourceType)
	}
}

func TestListResourcesByNameAndType(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add some resources to the database.
	r1 := addResource(db, schema, "r1", "app")
	addResource(db, schema, "r2", "app")
	addResource(db, schema, "r1", "analysis")
	addResource(db, schema, "r2", "analysis")

	// List the resources and verify we get the expected number of results.
	result := listResources(db, schema, r1.ResourceType, r1.Name)
	if len(result.Resources) != 1 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}

	// Verify that we got the expected result.
	resource := result.Resources[0]
	if *resource.Name != *r1.Name {
		t.Errorf("unexpected resource name: %s", *resource.Name)
	}
	if *resource.ResourceType != *r1.ResourceType {
		t.Errorf("unexpected resource type: %s", *resource.ResourceType)
	}
}

func TestListResourcesEmpty(t *testing.T) {
	if !shouldRun() {
		return
	}

	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource to the database.
	result := listResources(db, schema, nil, nil)
	if result.Resources == nil {
		t.Errorf("recieved a nil resource list")
	}
}

func TestUpdateResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource to the database.
	r2 := addResource(db, schema, "r2", "app")

	// Update the resource and verify the result,
	d2 := updateResource(db, schema, *r2.ID, "d2")
	if *d2.Name != "d2" {
		t.Errorf("unexpected resource name: %s", *d2.Name)
	}
	if *d2.ResourceType != "app" {
		t.Errorf("unexpected resource type: %s", *d2.ResourceType)
	}

	// List the resources and verify that we get the expected number of results.
	result := listResources(db, schema, nil, nil)
	if len(result.Resources) != 1 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}

	// Verify that we got the expected result.
	resource := result.Resources[0]
	if *resource.Name != *d2.Name {
		t.Errorf("unexpected resource name listed: %s", *resource.Name)
	}
	if *resource.ResourceType != *d2.ResourceType {
		t.Errorf("unexpected resource type listed: %s", *resource.ResourceType)
	}
}

func TestUpdateNonExistentResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to update a non-existent resource.
	responder := updateResourceAttempt(db, schema, FakeID, "foo")

	// Verify that we got the expected result.
	errorOut := responder.(*resources.UpdateResourceNotFound).Payload
	expected := fmt.Sprintf("resource, %s, not found", FakeID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure message: %s", *errorOut.Reason)
	}
}

func TestUpdateResourceDuplicateName(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add two resources to the database.
	r1 := addResource(db, schema, "r1", "app")
	r2 := addResource(db, schema, "r2", "app")

	// Attempt to give the second resource the first one's name.
	responder := updateResourceAttempt(db, schema, *r2.ID, *r1.Name)

	// Verify that we got the expected result.
	errorOut := responder.(*resources.UpdateResourceBadRequest).Payload
	expected := fmt.Sprintf("a resource of the same type named, '%s', already exists", *r1.Name)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure message: %s", *errorOut.Reason)
	}
}

func TestDeleteResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource to the database then delete it.
	r1 := addResource(db, schema, "r1", "app")
	deleteResource(db, schema, *r1.ID)

	// Verify that the resource was deleted.
	result := listResources(db, schema, nil, nil)
	if len(result.Resources) != 0 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}
}

func TestDeleteResourceByName(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Add a resource to the database then delete it.
	r1 := addResource(db, schema, "r1", "app")
	deleteResourceByName(db, schema, *r1.ResourceType, *r1.Name)

	// Verify that the resource was deleted.
	result := listResources(db, schema, nil, nil)
	if len(result.Resources) != 0 {
		t.Fatalf("unexpected number of resources listed: %d", len(result.Resources))
	}
}

func TestDeleteNonExistentResource(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to delete a non-existent resource.
	responder := deleteResourceAttempt(db, schema, FakeID)

	// Verify that we got the expected result.
	errorOut := responder.(*resources.DeleteResourceNotFound).Payload
	expected := fmt.Sprintf("resource, %s, not found", FakeID)
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure message: %s", *errorOut.Reason)
	}
}

func TestDeleteNonExistentResourceByName(t *testing.T) {
	if !shouldRun() {
		return
	}

	// Initialize the database.
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)

	// Attempt to delete a non-existent resource.
	responder := deleteResourceByNameAttempt(db, schema, "foo", "bar")

	// Verify that we got the expected result.
	errorOut := responder.(*resources.DeleteResourceByNameNotFound).Payload
	expected := "resource not found: foo:bar"
	if *errorOut.Reason != expected {
		t.Errorf("unexpected failure message: %s", *errorOut.Reason)
	}
}
