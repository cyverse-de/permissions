package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/models"
)

func rowsToResourceList(rows *sql.Rows) ([]*models.ResourceOut, error) {

	// Get the resources.
	resources := make([]*models.ResourceOut, 0)
	for rows.Next() {
		var resource models.ResourceOut
		if err := rows.Scan(&resource.ID, &resource.Name, &resource.ResourceType); err != nil {
			return nil, err
		}
		resources = append(resources, &resource)
	}

	return resources, nil
}

func rowsToResource(rows *sql.Rows, duplicateErr error) (*models.ResourceOut, error) {

	// Get the resources.
	resources, err := rowsToResourceList(rows)
	if err != nil {
		return nil, err
	}

	// Check for duplicates. This shouldn't happen unless there's a bug in the query.
	if len(resources) > 1 {
		return nil, duplicateErr
	}

	// Return the result.
	if len(resources) < 1 {
		return nil, nil
	}
	return resources[0], nil
}

func rowToResource(row *sql.Row) (*models.ResourceOut, error) {
	var resource models.ResourceOut
	if err := row.Scan(&resource.ID, &resource.Name, &resource.ResourceType); err != nil {
		return nil, err
	}
	return &resource, nil
}

// CountResourcesOfType counts the number of resources of the given type.
func CountResourcesOfType(ctx context.Context, tx *sql.Tx, resourceTypeID *string) (int64, error) {

	// Query the database.
	query := "SELECT count(*) FROM resources WHERE resource_type_id = $1"
	row := tx.QueryRowContext(ctx, query, resourceTypeID)

	// Return the result.
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ResourceExists determines whether or not the resource with the given ID exists.
func ResourceExists(ctx context.Context, tx *sql.Tx, id *string) (bool, error) {

	// Query the database.
	query := "SELECT count(*) FROM resources WHERE id = $1"
	row := tx.QueryRowContext(ctx, query, id)

	// Get the result.
	var count uint32
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetResourceByName obtains information about all resources with the given name. Multiple resources may have the same
// name as long as the types are different.
func GetResourceByName(ctx context.Context, tx *sql.Tx, name *string, resourceTypeID *string) (*models.ResourceOut, error) {

	// Query the database.
	query := `SELECT r.id, r.name, t.name AS resource_type
            FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
            WHERE t.id = $1 and r.name = $2`
	rows, err := tx.QueryContext(ctx, query, resourceTypeID, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get the resource.
	return rowsToResource(rows, fmt.Errorf("found multiple resources of the same type named, '%s'", *name))
}

// GetResourceByNameAndType obtains information about the resource with the given name and type.
func GetResourceByNameAndType(ctx context.Context, tx *sql.Tx, name, resourceTypeName string) (*models.ResourceOut, error) {

	// Query the database.
	query := `SELECT r.id, r.name, t.name AS resource_type
            FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
            WHERE t.name = $1 and r.name = $2`
	rows, err := tx.QueryContext(ctx, query, resourceTypeName, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get the resource.
	duplicateErr := fmt.Errorf("found multiple resources with the same type and name: %s:%s", resourceTypeName, name)
	return rowsToResource(rows, duplicateErr)
}

// GetDuplicateResourceByName obtains information about duplicate resources in the database.
func GetDuplicateResourceByName(ctx context.Context, tx *sql.Tx, id *string, name *string) (*models.ResourceOut, error) {

	// Query the database.
	query := `SELECT r.id, r.name, t.name AS resource_type
            FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
            WHERE r.id != $1
            AND r.name = $2
            AND r.resource_type_id = (SELECT resource_type_id FROM resources WHERE id = $1)`
	rows, err := tx.QueryContext(ctx, query, id, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get the resource.
	return rowsToResource(rows, fmt.Errorf("found multiple resources of the same type named, '%s'", *name))
}

// AddResource adds a resource to the database.
func AddResource(ctx context.Context, tx *sql.Tx, name *string, resourceTypeID *string) (*models.ResourceOut, error) {

	// Update the database.
	query := `INSERT INTO resources (name, resource_type_id) VALUES ($1, $2)
            RETURNING id, name, (SELECT name FROM resource_types t WHERE t.id = resource_type_id)`
	row := tx.QueryRowContext(ctx, query, name, resourceTypeID)

	// Return the result.
	return rowToResource(row)
}

// UpdateResource updates a resource in the database.
func UpdateResource(ctx context.Context, tx *sql.Tx, id *string, name *string) (*models.ResourceOut, error) {

	// Update the database.
	query := `UPDATE resources SET name = $1 WHERE id = $2
            RETURNING id, name, (SELECT name FROM resource_types t WHERE t.id = resource_type_id)`
	row := tx.QueryRowContext(ctx, query, name, id)

	// Return the result.
	return rowToResource(row)
}

// ListResources lists resources in the database, optionally filtering by resource type and resource name.
func ListResources(ctx context.Context, tx *sql.Tx, resourceTypeName, resourceName *string) ([]*models.ResourceOut, error) {

	// Query the database.
	var rows *sql.Rows
	var err error
	if resourceTypeName != nil && resourceName != nil {
		query := `SELECT r.id, r.name, t.name AS resource_type
              FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
              WHERE t.name = $1 AND r.name = $2`
		rows, err = tx.QueryContext(ctx, query, *resourceTypeName, *resourceName)
	} else if resourceTypeName != nil {
		query := `SELECT r.id, r.name, t.name AS resource_type
              FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
              WHERE t.name = $1`
		rows, err = tx.QueryContext(ctx, query, *resourceTypeName)
	} else if resourceName != nil {
		query := `SELECT r.id, r.name, t.name AS resource_type
              FROM resources r JOIN resource_types t ON r.resource_type_id = t.id
              WHERE r.name = $1`
		rows, err = tx.QueryContext(ctx, query, *resourceName)
	} else {
		query := `SELECT r.id, r.name, t.name AS resource_type
            FROM resources r JOIN resource_types t ON r.resource_type_id = t.id`
		rows, err = tx.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the list of resources.
	return rowsToResourceList(rows)
}

// DeleteResource removes a resource from the database.
func DeleteResource(ctx context.Context, tx *sql.Tx, id *string) error {

	// Update the database.
	stmt := "DELETE FROM resources WHERE id = $1"
	result, err := tx.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	// Verify that a row was deleted.
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no resources deleted for id %s", *id)
	}
	if count > 1 {
		return fmt.Errorf("multiple resources deleted for id %s", *id)
	}

	return nil
}
