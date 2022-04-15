package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/models"
)

// ListResourceTypes lists all defined resource types.
func ListResourceTypes(ctx context.Context, tx *sql.Tx, resourceTypeName *string) ([]*models.ResourceTypeOut, error) {

	// Query the database.
	var rows *sql.Rows
	var err error
	if resourceTypeName == nil {
		query := "SELECT id, name, description FROM resource_types"
		rows, err = tx.QueryContext(ctx, query)
	} else {
		query := "SELECT id, name, description FROM resource_types WHERE name = $1"
		rows, err = tx.QueryContext(ctx, query, *resourceTypeName)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the list of resource types.
	resourceTypes := make([]*models.ResourceTypeOut, 0)
	for rows.Next() {
		var resourceType models.ResourceTypeOut
		if err := rows.Scan(&resourceType.ID, &resourceType.Name, &resourceType.Description); err != nil {
			return nil, err
		}
		resourceTypes = append(resourceTypes, &resourceType)
	}

	// Check for any uncaught errors.
	if err := rows.Err(); err != nil {
		return resourceTypes, err
	}

	return resourceTypes, nil
}

// GetResourceTypeByName gets information about the resource type with the given name.
func GetResourceTypeByName(ctx context.Context, tx *sql.Tx, name *string) (*models.ResourceTypeOut, error) {

	// Query the database.
	query := `SELECT id, name, description FROM resource_types
	          WHERE lower(trim(regexp_replace(name, '\s+', ' ', 'g')))
	              = lower(trim(regexp_replace($1, '\s+', ' ', 'g')))`
	rows, err := tx.QueryContext(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get the resource type.
	resourceTypes := make([]*models.ResourceTypeOut, 0)
	for rows.Next() {
		var resourceType models.ResourceTypeOut
		if err := rows.Scan(&resourceType.ID, &resourceType.Name, &resourceType.Description); err != nil {
			return nil, err
		}
		resourceTypes = append(resourceTypes, &resourceType)
	}

	// Check for duplicates. There's a uniqueness constraint on the database so this shouldn't happen.
	if len(resourceTypes) > 1 {
		return nil, fmt.Errorf("found multiple resource types with the name: %s", *name)
	}

	// Return the result.
	if len(resourceTypes) < 1 {
		return nil, nil
	}
	return resourceTypes[0], nil
}

// GetDuplicateResourceTypeByName returns information about the resource type with the given name, and checks for
// duplicates
func GetDuplicateResourceTypeByName(ctx context.Context, tx *sql.Tx, id *string, name *string) (*models.ResourceTypeOut, error) {

	// Query the database.
	query := `SELECT id, name, description FROM resource_types
	          WHERE id != $1
	          AND lower(trim(regexp_replace(name, '\s+', ' ', 'g')))
	            = lower(trim(regexp_replace($2, '\s+', ' ', 'g')))`
	rows, err := tx.QueryContext(ctx, query, id, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get the resource type.
	resourceTypes := make([]*models.ResourceTypeOut, 0)
	for rows.Next() {
		var resourceType models.ResourceTypeOut
		if err := rows.Scan(&resourceType.ID, &resourceType.Name, &resourceType.Description); err != nil {
			return nil, err
		}
		resourceTypes = append(resourceTypes, &resourceType)
	}

	// Check for duplicates. There's a uniqueness constraint on the database so this shouldn't happen.
	if len(resourceTypes) > 1 {
		return nil, fmt.Errorf("found multiple duplicate resource types with the name: %s", *name)
	}

	// Return the result.
	if len(resourceTypes) < 1 {
		return nil, nil
	}
	return resourceTypes[0], nil
}

// ResourceTypeExists determines whether or not the resource type with the given ID exists.
func ResourceTypeExists(ctx context.Context, tx *sql.Tx, id *string) (bool, error) {

	// Query the database.
	query := "SELECT count(*) FROM resource_types WHERE id = $1"
	row := tx.QueryRowContext(ctx, query, id)

	// Get the result.
	var count uint32
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddNewResourceType adds a new resource type to the database.
func AddNewResourceType(ctx context.Context, tx *sql.Tx, resourceTypeIn *models.ResourceTypeIn) (*models.ResourceTypeOut, error) {

	// Insert the resource type.
	query := `INSERT INTO resource_types (name, description)
	          VALUES (trim(regexp_replace($1, '\s+', ' ', 'g')), $2)
	          RETURNING id, name, description`
	row := tx.QueryRowContext(ctx, query, resourceTypeIn.Name, resourceTypeIn.Description)

	// Get the newly created resource type.
	var resourceTypeOut models.ResourceTypeOut
	if err := row.Scan(&resourceTypeOut.ID, &resourceTypeOut.Name, &resourceTypeOut.Description); err != nil {
		return nil, err
	}
	return &resourceTypeOut, nil
}

// UpdateResourceType updates a resource type in the database.
func UpdateResourceType(
	ctx context.Context,
	tx *sql.Tx,
	id *string,
	resourceTypeIn *models.ResourceTypeIn,
) (*models.ResourceTypeOut, error) {

	// Update the databse.
	statement := `UPDATE resource_types
	              SET name = trim(regexp_replace($1, '\s+', ' ', 'g')),
	                  description = $2
	              WHERE id = $3
	              RETURNING id, name, description`
	row := tx.QueryRowContext(ctx, statement, resourceTypeIn.Name, resourceTypeIn.Description, id)

	// Get the newly updated resource type.
	var resourceTypeOut models.ResourceTypeOut
	if err := row.Scan(&resourceTypeOut.ID, &resourceTypeOut.Name, &resourceTypeOut.Description); err != nil {
		return nil, err
	}

	return &resourceTypeOut, nil
}

// DeleteResourceType removes a resource type from the database.
func DeleteResourceType(ctx context.Context, tx *sql.Tx, id *string) error {

	// Update the database.
	statement := "DELETE FROM resource_types WHERE id = $1"
	result, err := tx.ExecContext(ctx, statement, id)
	if err != nil {
		return err
	}

	// Verify that a row was deleted.
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no resource types deleted for id %s", *id)
	}
	if count > 1 {
		return fmt.Errorf("multiple resource types deleted for id %s", *id)
	}

	return nil
}
