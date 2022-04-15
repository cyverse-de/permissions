package db

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/cyverse-de/permissions/models"
)

// rowsToPermissionList returns a list of permissions for the given result set. The columns in the reult set
// must be permission ID, internal subject ID, external subject ID, subject type, resource ID, resource name,
// resource type, and permission level, in that order.
func rowsToPermissionList(rows *sql.Rows) ([]*models.Permission, error) {

	// Build the list of permissions.
	permissions := make([]*models.Permission, 0)
	for rows.Next() {
		var dto PermissionDTO
		err := rows.Scan(
			&dto.ID, &dto.InternalSubjectID, &dto.SubjectID, &dto.SubjectType, &dto.ResourceID,
			&dto.ResourceName, &dto.ResourceType, &dto.PermissionLevel,
		)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, dto.ToPermission())
	}

	return permissions, nil
}

// rowsToAbbreviatedPermissionList returns a list of permissions for the given result set. The columns in the
// result set must be the permission ID, resource name, resource type, and permission level, in that order.
func rowsToAbbreviatedPermissionList(rows *sql.Rows) ([]*models.AbbreviatedPermission, error) {

	// Build the list of abbreviated permissions.
	permissions := make([]*models.AbbreviatedPermission, 0)
	for rows.Next() {
		var permission models.AbbreviatedPermission
		err := rows.Scan(
			&permission.ID, &permission.ResourceName, &permission.ResourceType, &permission.PermissionLevel,
		)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, &permission)
	}

	return permissions, nil
}

// ListPermissions lists all existing permissions.
func ListPermissions(ctx context.Context, tx *sql.Tx) ([]*models.Permission, error) {

	// Query the database.
	query := `SELECT p.id AS id,
	                 s.id AS internal_subject_id,
	                 s.subject_id AS subject_id,
	                 s.subject_type AS subject_type,
	                 r.id AS resource_id,
	                 r.name AS resource_name,
	                 rt.name AS resource_type,
	                 pl.name AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          ORDER BY s.subject_id, r.name, pl.precedence`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// ListResourcePermissions lists permissions associated with a specific resource.
func ListResourcePermissions(ctx context.Context, tx *sql.Tx, resourceTypeName, resourceName string) ([]*models.Permission, error) {

	// Query the database.
	query := `SELECT p.id AS id,
	                 s.id AS internal_subject_id,
	                 s.subject_id AS subject_id,
	                 s.subject_type AS subject_type,
	                 r.id AS resource_id,
	                 r.name AS resource_name,
	                 rt.name AS resource_type,
	                 pl.name AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
            WHERE rt.name = $1 AND r.name = $2
	          ORDER BY s.subject_id`
	rows, err := tx.QueryContext(ctx, query, resourceTypeName, resourceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// PermissionsForSubjects lists permissions granted to zero or more subjects.
func PermissionsForSubjects(ctx context.Context, tx *sql.Tx, subjectIds []string) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectsMinLevel lists permissions of at least the given level granted to zero or more subjects.
func PermissionsForSubjectsMinLevel(ctx context.Context, tx *sql.Tx, subjectIds []string, minLevel string) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $2)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa, minLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectsAndResourceType lists permissions that have been granted to zero or more subjects for the
// specified type of resource.
func PermissionsForSubjectsAndResourceType(
	ctx context.Context, tx *sql.Tx, subjectIds []string, resourceTypeName string,
) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
            AND rt.Name = $2
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa, resourceTypeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectsAndResourceTypeMinLevel lists permissions of at least the minimum level that have been
// granted to zero or more subjects for the specified type of resource.
func PermissionsForSubjectsAndResourceTypeMinLevel(
	ctx context.Context, tx *sql.Tx, subjectIds []string, resourceTypeName, minLevel string,
) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
            AND rt.Name = $2
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $3)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa, resourceTypeName, minLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// permissionLevelPrecedenceExpression returns a SelectBuilder representing a permission level precedence
// expression that can be used in a where clause.
func permissionLevelPrecedenceExpression(prefix, permissionLevel string) sq.SelectBuilder {
	return psql.Select("precedence").
		Prefix(fmt.Sprintf("%s (", prefix)).
		From("permission_levels").
		Where(sq.Eq{"name": permissionLevel}).
		Suffix(")")
}

// AbbreviatedPermissionsForSubjectAndResourceType lists permissions for a subject and resource type. If the
// minLevel parameter is specified, permissions that don't meet or exceed the minimum level will be omitted
// from the results.
func AbbreviatedPermissionsForSubjectAndResourceType(
	ctx context.Context, tx *sql.Tx, subjectIDs []string, resourceTypeName string, minLevel *string,
) ([]*models.AbbreviatedPermission, error) {

	// Begin building the query.
	builder := psql.Select(
		"first_value(p.id) OVER w AS id",
		"first_value(r.name) OVER w AS resource_name",
		"first_value(rt.name) OVER w AS resource_type",
		"first_value(pl.name) OVER w AS permission_level",
	).Distinct().Options("ON (r.id)").
		From("permissions p").
		Join("permission_levels pl ON p.permission_level_id = pl.id").
		Join("subjects s ON p.subject_id = s.id").
		Join("resources r ON p.resource_id = r.id").
		Join("resource_types rt ON r.resource_type_id = rt.id").
		Where(sq.Eq{"rt.name": resourceTypeName}).
		Where(sq.Eq{"s.subject_id": subjectIDs})

	// Add the permission level expression if a minimum level was specified.
	if minLevel != nil {
		builder = builder.Where(permissionLevelPrecedenceExpression("pl.precedence <=", *minLevel))
	}

	// Add the window and the ORDER BY clause. The ORDER BY clause has to appear here because Squirrel doesn't have
	// explicit support for the WINDOW clause.
	builder = builder.Suffix("WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence) ORDER BY r.id")

	// Generate the query.
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	// Execute the query.
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToAbbreviatedPermissionList(rows)
}

// PermissionsForSubjectsAndResource lists permissions granted to zero or more subjects for a specific resource.
func PermissionsForSubjectsAndResource(
	ctx context.Context, tx *sql.Tx, subjectIds []string, resourceTypeName, resourceName string,
) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
            AND rt.name = $2
	          AND r.name = $3
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa, resourceTypeName, resourceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectsAndResourceMinLevel lists permissions of at least the minimum level that have been granted
// to zero or more subjects for a specific resource.
func PermissionsForSubjectsAndResourceMinLevel(
	ctx context.Context, tx *sql.Tx, subjectIds []string, resourceTypeName, resourceName, minLevel string,
) ([]*models.Permission, error) {
	sa := StringArray(subjectIds)

	// Query the database.
	query := `SELECT DISTINCT ON (r.id)
	              first_value(p.id) OVER w AS id,
	              first_value(s.id) OVER w AS internal_subject_id,
	              first_value(s.subject_id) OVER w AS subject_id,
	              first_value(s.subject_type) OVER w AS subject_type,
	              r.id AS resource_id,
	              first_value(r.name) OVER w AS resource_name,
	              first_value(rt.name) OVER w AS resource_type,
	              first_value(pl.name) OVER w AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE s.subject_id = any($1)
            AND rt.name = $2
	          AND r.name = $3
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $4)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, &sa, resourceTypeName, resourceName, minLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPermissionList(rows)
}

// GetPermissionByID obtains information about a specific permission.
func GetPermissionByID(ctx context.Context, tx *sql.Tx, permissionID string) (*models.Permission, error) {

	// Query the database.
	query := `SELECT p.id AS id,
	                 s.id AS internal_subject_id,
	                 s.subject_id AS subject_id,
	                 s.subject_type AS subject_type,
	                 r.id AS resource_id,
	                 r.name AS resource_name,
	                 rt.name AS resource_type,
	                 pl.name AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE p.id = $1`
	rows, err := tx.QueryContext(ctx, query, &permissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the list of permissions.
	permissions, err := rowsToPermissionList(rows)
	if err != nil {
		return nil, err
	}

	// Check for duplicates. This shouldn't happen because of the primary key constraint.
	if len(permissions) > 1 {
		return nil, fmt.Errorf("multiple permissions found for ID: %s", permissionID)
	}

	// Return the result.
	if len(permissions) < 1 {
		return nil, nil
	}
	return permissions[0], nil
}

// GetPermissionLevelIDByName returns the identifier for the permission level with the given name.
func GetPermissionLevelIDByName(ctx context.Context, tx *sql.Tx, level models.PermissionLevel) (*string, error) {

	// Query the database.
	query := "SELECT id FROM permission_levels WHERE name = $1"
	rows, err := tx.QueryContext(ctx, query, string(level))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the list of permission levels.
	ids := make([]*string, 0)
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, &id)
	}

	// Check for duplicate results. This shouldn't happen because there's a uniqueness constraint.
	if len(ids) > 1 {
		return nil, fmt.Errorf("duplicate permission levels found: %s", string(level))
	}

	// Return the result.
	if len(ids) < 1 {
		return nil, nil
	}
	return ids[0], nil
}

// UpsertPermission updates a permission or inserts it if it doesn't exist.
func UpsertPermission(
	ctx context.Context,
	tx *sql.Tx,
	subjectID models.InternalSubjectID,
	resourceID string,
	permissionLevelID string,
) (*models.Permission, error) {

	// Update the database.
	stmt := `INSERT INTO permissions (subject_id, resource_id, permission_level_id) VALUES ($1, $2, $3)
	         ON CONFLICT (subject_id, resource_id) DO UPDATE
	         SET permission_level_id = EXCLUDED.permission_level_id
	         RETURNING id`
	row := tx.QueryRowContext(ctx, stmt, string(subjectID), resourceID, permissionLevelID)

	// Extract the permission ID.
	var permissionID string
	if err := row.Scan(&permissionID); err != nil {
		return nil, err
	}

	// Look up the permission.
	permission, err := GetPermissionByID(ctx, tx, permissionID)
	if err != nil {
		return nil, err
	} else if permission == nil {
		return nil, fmt.Errorf("unable to look up permission after upsert: %s", permissionID)
	}
	return permission, nil
}

// GetPermission gets a subject's permission to a specific resource if it exists.
func GetPermission(
	ctx context.Context,
	tx *sql.Tx,
	subjectID models.InternalSubjectID,
	resourceID string,
) (*models.Permission, error) {

	// Query the database.
	query := `SELECT p.id AS id,
	                 s.id AS internal_subject_id,
	                 s.subject_id AS subject_id,
	                 s.subject_type AS subject_type,
	                 r.id AS resource_id,
	                 r.name AS resource_name,
	                 rt.name AS resource_type,
	                 pl.name AS permission_level
	          FROM permissions p
	          JOIN permission_levels pl ON p.permission_level_id = pl.id
	          JOIN subjects s ON p.subject_id = s.id
	          JOIN resources r ON p.resource_id = r.id
	          JOIN resource_types rt ON r.resource_type_id = rt.id
	          WHERE p.subject_id = $1
	          AND p.resource_id = $2`
	rows, err := tx.QueryContext(ctx, query, string(subjectID), resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the list of permissions.
	permissions, err := rowsToPermissionList(rows)
	if err != nil {
		return nil, err
	}

	// Check for duplicates. This shouldn't happen because of the uniqueness constraint.
	if len(permissions) > 1 {
		return nil, fmt.Errorf("multiple permissions found for subject/resource: %s/%s", subjectID, resourceID)
	}

	// Return the result.
	if len(permissions) < 1 {
		return nil, nil
	}
	return permissions[0], nil
}

// DeletePermission removes a permission from the database.
func DeletePermission(ctx context.Context, tx *sql.Tx, id models.PermissionID) error {

	// Update the database.
	stmt := "DELETE FROM permissions WHERE id = $1"
	result, err := tx.ExecContext(ctx, stmt, string(id))
	if err != nil {
		return err
	}

	// Verify that a row was deleted.
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no permissions deleted for id %s", id)
	}
	if count > 1 {
		return fmt.Errorf("multiple permissions deleted for id %s", id)
	}

	return nil
}

// CopyPermissions copies permissions from one subject to another.
func CopyPermissions(ctx context.Context, tx *sql.Tx, source, dest *models.SubjectOut) error {

	// Copy or update permissions.
	stmt := `INSERT INTO permissions AS d (subject_id, resource_id, permission_level_id)
           SELECT $2, resource_id, permission_level_id
           FROM permissions WHERE subject_id = $1
           ON CONFLICT (subject_id, resource_id) DO UPDATE SET permission_level_id = (
               SELECT id FROM permission_levels
               WHERE id IN (d.permission_level_id, EXCLUDED.permission_level_id)
               ORDER BY precedence LIMIT 1
           )`
	_, err := tx.ExecContext(ctx, stmt, &source.ID, &dest.ID)

	return err
}
