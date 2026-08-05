package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/cyverse-de/permissions/logger"
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

// subjectScope builds the clauses that restrict a permission query to one
// subject, optionally expanded to the groups that subject belongs to.
//
// The expansion reads group_effective_members, the materialized transitive
// closure, so it stays a plain join no matter how deeply groups nest. It
// replaces a call to another service whose result was fed back in as a literal
// array.
//
// Two shapes rather than one query with a conditional arm: this is the query
// behind every authorization decision in the DE, and swapping a literal array of
// known cardinality for a CTE can flip a nested-loop index scan into a hash join
// over the whole permissions table. Keeping the non-expanding form free of the
// CTE lets the planner see it unchanged.
//
// $1 is the external subject identifier in both shapes, so the remaining
// parameter numbers are unaffected.
func subjectScope(lookup bool) (prefix, where string) {
	if !lookup {
		return "", "s.subject_id = $1"
	}

	// A group-typed requester is safe here only because
	// group_effective_members.member_type is CHECK-pinned to 'user' in the
	// de-database schema, so a group can never appear as a member_id and the
	// expansion adds nothing to it. Relaxing that constraint would change
	// authorization results.
	prefix = `WITH requester AS (
	              SELECT id FROM subjects WHERE subject_id = $1
	          ),
	          subject_ids AS (
	              SELECT id FROM requester
	              UNION
	              SELECT em.group_id
	                FROM group_effective_members em
	                JOIN requester r ON em.member_id = r.id
	          )
	          `
	return prefix, "p.subject_id IN (SELECT id FROM subject_ids)"
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
	defer logger.LogClose(rows, "permissions rows")

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
	defer logger.LogClose(rows, "permissions rows")

	return rowsToPermissionList(rows)
}

// PermissionsForSubject lists permissions granted to a subject, expanded to the
// subject's groups when lookup is true.
func PermissionsForSubject(ctx context.Context, tx *sql.Tx, subjectID string, lookup bool) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectMinLevel lists permissions of at least the given level granted to a subject, expanded
// to the subject's groups when lookup is true.
func PermissionsForSubjectMinLevel(ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, minLevel string) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $2)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID, minLevel)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectAndResourceType lists permissions that have been granted to a subject for the specified
// type of resource, expanded to the subject's groups when lookup is true.
func PermissionsForSubjectAndResourceType(
	ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, resourceTypeName string,
) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
            AND rt.Name = $2
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID, resourceTypeName)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectAndResourceTypeMinLevel lists permissions of at least the minimum level that have been
// granted to a subject for the specified type of resource, expanded to the subject's groups when lookup is true.
func PermissionsForSubjectAndResourceTypeMinLevel(
	ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, resourceTypeName, minLevel string,
) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
            AND rt.Name = $2
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $3)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID, resourceTypeName, minLevel)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

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
	ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, resourceTypeName string, minLevel *string,
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
		Where(sq.Eq{"rt.name": resourceTypeName})

	// Restrict to the subject, expanded to its groups when asked. Squirrel
	// numbers placeholders itself, so the scope is applied as an expression
	// rather than by splicing the shared SQL used by the other queries.
	if lookup {
		builder = builder.Where(sq.Expr(
			`p.subject_id IN (
			     WITH requester AS (SELECT id FROM subjects WHERE subject_id = ?)
			     SELECT id FROM requester
			     UNION
			     SELECT em.group_id FROM group_effective_members em JOIN requester r ON em.member_id = r.id
			 )`, subjectID))
	} else {
		builder = builder.Where(sq.Eq{"s.subject_id": subjectID})
	}

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
	defer logger.LogClose(rows, "permissions rows")

	return rowsToAbbreviatedPermissionList(rows)
}

// PermissionsForSubjectAndResource lists permissions granted to a subject for a specific resource, expanded to
// the subject's groups when lookup is true.
func PermissionsForSubjectAndResource(
	ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, resourceTypeName, resourceName string,
) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
            AND rt.name = $2
	          AND r.name = $3
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID, resourceTypeName, resourceName)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

	return rowsToPermissionList(rows)
}

// PermissionsForSubjectAndResourceMinLevel lists permissions of at least the minimum level that have been
// granted to a subject for a specific resource, expanded to the subject's groups when lookup is true.
func PermissionsForSubjectAndResourceMinLevel(
	ctx context.Context, tx *sql.Tx, subjectID string, lookup bool, resourceTypeName, resourceName, minLevel string,
) ([]*models.Permission, error) {
	prefix, subjectWhere := subjectScope(lookup)

	// Query the database.
	query := prefix + `SELECT DISTINCT ON (r.id)
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
	          WHERE ` + subjectWhere + `
            AND rt.name = $2
	          AND r.name = $3
            AND pl.precedence <= (SELECT precedence FROM permission_levels WHERE name = $4)
	          WINDOW w AS (PARTITION BY r.id ORDER BY pl.precedence)
            ORDER BY r.id`
	rows, err := tx.QueryContext(ctx, query, subjectID, resourceTypeName, resourceName, minLevel)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "permissions rows")

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
	defer logger.LogClose(rows, "permissions rows")

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
	defer logger.LogClose(rows, "permissions rows")

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
	defer logger.LogClose(rows, "permissions rows")

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

// ErrTooManySubjects reports that expanding a resource's group permissions would
// produce more subjects than the caller allowed.
var ErrTooManySubjects = errors.New("too many subjects after group expansion")

// ExpandedResourcePermissions lists a resource's permissions with each group
// permission replaced by the group's effective members, keeping the strongest
// level per subject.
//
// This replaces a loop that asked another service for each group's membership in
// turn, then created a subject row for any member it had never seen -- writing
// to the database while serving a GET. Effective members are already subjects
// here, so nothing needs creating and the whole expansion is one statement.
//
// maxSubjects bounds the work as well as the result. `de-users` contains
// essentially every DE user, so expanding a public resource's permissions is
// unbounded work; it is slow enough over HTTP today that nobody does it, and
// making it fast is exactly what would invite someone to. A membership count
// runs first so that an over-cap request is refused for the price of an index
// scan instead of the full expansion and sort it is refusing to do; the LIMIT
// on the expansion itself remains as a backstop. Exceeding the bound is an
// error rather than a truncation, because a silently short list of who can see
// a resource is worse than a refusal.
func ExpandedResourcePermissions(
	ctx context.Context, tx *sql.Tx, resourceTypeName, resourceName string, maxSubjects int,
) ([]*models.Permission, error) {

	// The count reads only (group_id, member_id) from group_effective_members so
	// it can be answered from the primary key alone. It may overcount -- a user
	// holding both a direct grant and a group membership is counted twice -- but
	// never undercounts, so it can only refuse expansions already near the cap,
	// and refusing early is the point.
	countQuery := `WITH matching AS (
	              SELECT p.subject_id
	                FROM permissions p
	                JOIN resources r ON r.id = p.resource_id
	                JOIN resource_types rt ON rt.id = r.resource_type_id
	               WHERE rt.name = $1 AND r.name = $2
	          )
	          SELECT (SELECT count(*)
	                    FROM matching m
	                    JOIN subjects s ON s.id = m.subject_id
	                   WHERE s.subject_type = 'user')
	               + (SELECT count(DISTINCT em.member_id)
	                    FROM matching m
	                    JOIN subjects gs ON gs.id = m.subject_id AND gs.subject_type = 'group'
	                    JOIN group_effective_members em ON em.group_id = gs.id)`
	var subjectCount int
	if err := tx.QueryRowContext(ctx, countQuery, resourceTypeName, resourceName).Scan(&subjectCount); err != nil {
		return nil, err
	}
	if subjectCount > maxSubjects {
		return nil, fmt.Errorf("%w: %s/%s expands to more than %d subjects",
			ErrTooManySubjects, resourceTypeName, resourceName, maxSubjects)
	}

	query := `WITH matching AS (
	              SELECT p.id, p.subject_id, p.permission_level_id,
	                     r.id AS resource_id, r.name AS resource_name, rt.name AS resource_type
	                FROM permissions p
	                JOIN resources r ON r.id = p.resource_id
	                JOIN resource_types rt ON rt.id = r.resource_type_id
	               WHERE rt.name = $1 AND r.name = $2
	          ),
	          expanded AS (
	              SELECT m.id, s.id AS internal_subject_id, s.subject_id, s.subject_type,
	                     m.permission_level_id, m.resource_id, m.resource_name, m.resource_type
	                FROM matching m
	                JOIN subjects s ON s.id = m.subject_id
	               WHERE s.subject_type = 'user'
	              UNION ALL
	              SELECT m.id, ms.id, ms.subject_id, ms.subject_type,
	                     m.permission_level_id, m.resource_id, m.resource_name, m.resource_type
	                FROM matching m
	                JOIN subjects gs ON gs.id = m.subject_id AND gs.subject_type = 'group'
	                JOIN group_effective_members em ON em.group_id = gs.id
	                JOIN subjects ms ON ms.id = em.member_id
	          )
	          SELECT DISTINCT ON (e.internal_subject_id)
	                 e.id, e.internal_subject_id, e.subject_id, e.subject_type,
	                 e.resource_id, e.resource_name, e.resource_type, pl.name
	            FROM expanded e
	            JOIN permission_levels pl ON pl.id = e.permission_level_id
	           ORDER BY e.internal_subject_id, pl.precedence
	           LIMIT $3`

	// One more than the bound, so that exceeding it stays detectable even if
	// the pre-check ever disagrees with the expansion.
	rows, err := tx.QueryContext(ctx, query, resourceTypeName, resourceName, maxSubjects+1)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "expanded permissions rows")

	perms, err := rowsToPermissionList(rows)
	if err != nil {
		return nil, err
	}
	if len(perms) > maxSubjects {
		return nil, fmt.Errorf("%w: %s/%s expands to more than %d subjects",
			ErrTooManySubjects, resourceTypeName, resourceName, maxSubjects)
	}
	return perms, nil
}
