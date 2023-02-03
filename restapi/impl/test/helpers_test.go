// nolint
package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/cyverse-de/permissions/models"

	permsdb "github.com/cyverse-de/permissions/restapi/impl/db"
)

func addDefaultResourceType(tx *sql.Tx, name, description string, t *testing.T) {
	rt := &models.ResourceTypeIn{Name: &name, Description: description}
	if _, err := permsdb.AddNewResourceType(context.Background(), tx, rt); err != nil {
		tx.Rollback()
		t.Fatalf("unable to add default resource types: %s", err)
	}
}

func addDefaultResourceTypes(db *sql.DB, schema string, t *testing.T) {

	// Start a transaction.
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("unable to add default resource types: %s", err)
	}

	_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
	if err != nil {
		t.Fatalf("unable to add default resource types: %s", err)
	}

	// Add the default resource types.
	addDefaultResourceType(tx, "app", "app", t)
	addDefaultResourceType(tx, "analysis", "analysis", t)

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		t.Fatalf("unable to add default resource types: %s", err)
	}
}

func addTestResource(db *sql.DB, schema, name, resourceType string, t *testing.T) {

	// Start a Transaction.
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("unable to add a resource: %s", err)
	}

	_, err = tx.Exec(fmt.Sprintf("SET search_path TO %s", schema))
	if err != nil {
		t.Fatalf("unable to add a resource: %s", err)
	}

	// Get the resource type.
	rt, err := permsdb.GetResourceTypeByName(context.Background(), tx, &resourceType)
	if err != nil {
		tx.Rollback()
		t.Fatalf("unable to add a resource: %s", err)
	}
	if rt == nil {
		tx.Rollback()
		t.Fatalf("unable to add a resource: resource type not found: %s", resourceType)
	}

	// Insert the resource.
	if _, err := permsdb.AddResource(context.Background(), tx, &name, rt.ID); err != nil {
		tx.Rollback()
		t.Fatalf("unable to add a resource: %s", err)
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		t.Fatalf("unable to add a resource: %s", err)
	}
}

func newSubjectIn(subjectIDString, subjectTypeString string) *models.SubjectIn {
	subjectID := models.ExternalSubjectID(subjectIDString)
	subjectType := models.SubjectType(subjectTypeString)
	return &models.SubjectIn{
		SubjectID:   &subjectID,
		SubjectType: &subjectType,
	}
}

func newResourceIn(name, resourceType string) *models.ResourceIn {
	return &models.ResourceIn{
		Name:         &name,
		ResourceType: &resourceType,
	}
}
