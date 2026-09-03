package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
)

func rowsToSubjectList(rows *sql.Rows) ([]*models.SubjectOut, error) {

	// Get the list of subjects.
	subjects := make([]*models.SubjectOut, 0)
	for rows.Next() {
		var subjectDto SubjectDTO
		if err := rows.Scan(&subjectDto.ID, &subjectDto.SubjectID, &subjectDto.SubjectType); err != nil {
			return nil, err
		}
		subjects = append(subjects, subjectDto.ToSubjectOut())
	}

	// Check for any uncaught errors.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjects, nil
}

func rowsToSubject(rows *sql.Rows, duplicateErr error) (*models.SubjectOut, error) {

	// Get the list of subjects.
	subjects, err := rowsToSubjectList(rows)
	if err != nil {
		return nil, err
	}

	// Check for duplicates. This shouldn't happen unless there's a bug in the query.
	if len(subjects) > 1 {
		return nil, duplicateErr
	}

	// Return the result.
	if len(subjects) < 1 {
		return nil, nil
	}
	return subjects[0], nil
}

// AddSubject adds a subject to the database.
//
// A user subject is correlated with its DE user row when one exists, so that
// joining subjects to public.users does not mean reconstructing the suffixed
// username at every call site. The correlation is best-effort: a subject may be
// created before the user has ever logged in to the DE, and that must not
// prevent the subject from existing.
//
// The lookup is conditional on the subject type because subjects_user_id_is_user
// rejects a group row carrying a user_id, and guarded by NOT EXISTS because
// subjects_user_id_unique would turn a second claim on the same DE user into a
// failed insert rather than a missing correlation.
func AddSubject(
	ctx context.Context,
	tx *sql.Tx,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
	userSuffix string,
) (*models.SubjectOut, error) {

	// Update the database.
	// The subject type is passed a second time as a boolean rather than reused
	// in the WHERE clause: $2 is an enum in the column list, and comparing it to
	// a text literal makes PostgreSQL deduce two types for one parameter.
	query := `INSERT INTO subjects (subject_id, subject_type, user_id)
            VALUES ($1::text, $2, (
                SELECT u.id FROM public.users u
                 WHERE $4::boolean
                   AND u.username = $1::text || $3::text
                   AND NOT EXISTS (SELECT 1 FROM subjects o WHERE o.user_id = u.id)))
            RETURNING id, subject_id, subject_type, user_id`
	row := tx.QueryRowContext(ctx, query,
		string(subjectID), string(subjectType), userSuffix,
		subjectType == models.SubjectTypeUser)

	// Return the subject information.
	var subjectDto SubjectDTO
	var userID sql.NullString
	if err := row.Scan(&subjectDto.ID, &subjectDto.SubjectID, &subjectDto.SubjectType, &userID); err != nil {
		return nil, err
	}
	if subjectType == models.SubjectTypeUser && !userID.Valid {
		warnIfUserClaimed(ctx, tx, subjectID, userSuffix)
	}
	return subjectDto.ToSubjectOut(), nil
}

// warnIfUserClaimed distinguishes the two reasons a user subject can come back
// uncorrelated. No users row for the suffixed username is the common case --
// the user simply hasn't logged in to the DE yet -- and stays silent. A users
// row that exists anyway means the NOT EXISTS guard in AddSubject blocked
// because another subject already claims it, which is worth a warning: the new
// subject will never join to public.users and someone should look at which
// subject holds the correlation.
func warnIfUserClaimed(ctx context.Context, tx *sql.Tx, subjectID models.ExternalSubjectID, userSuffix string) {
	var userRowExists bool
	err := tx.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM public.users u WHERE u.username = $1::text || $2::text)",
		string(subjectID), userSuffix,
	).Scan(&userRowExists)
	if err != nil {
		logger.Log.Warnf("unable to check the DE user correlation for subject %s: %s", subjectID, err)
		return
	}
	if userRowExists {
		logger.Log.Warnf(
			"subject %s was created without a correlation to DE user %s%s even though that user exists; "+
				"this usually means another subject already claims the user's row",
			subjectID, subjectID, userSuffix)
	}
}

// UpdateSubject updates a subject in the database.
func UpdateSubject(
	ctx context.Context,
	tx *sql.Tx,
	id models.InternalSubjectID,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) (*models.SubjectOut, error) {

	// Update the database.
	query := `UPDATE subjects SET subject_id = $1, subject_type = $2
            WHERE id = $3
            RETURNING id, subject_id, subject_type`
	row := tx.QueryRowContext(ctx, query, string(subjectID), string(subjectType), string(id))

	// Return the subject information.
	var subjectDto SubjectDTO
	if err := row.Scan(&subjectDto.ID, &subjectDto.SubjectID, &subjectDto.SubjectType); err != nil {
		return nil, err
	}
	return subjectDto.ToSubjectOut(), nil
}

// SubjectIDExists determines whether or not the subject with the given external ID exists in the database.
func SubjectIDExists(ctx context.Context, tx *sql.Tx, subjectID models.ExternalSubjectID) (bool, error) {

	// Query the database.
	query := "SELECT count(*) FROM subjects WHERE subject_id = $1"
	row := tx.QueryRowContext(ctx, query, string(subjectID))

	// Get the result.
	var count uint32
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// SubjectExists determines whether or not the subject with the given internal ID exists in the database.
func SubjectExists(ctx context.Context, tx *sql.Tx, id models.InternalSubjectID) (bool, error) {

	// Query the database.
	query := "SELECT count(*) FROM subjects WHERE id = $1"
	row := tx.QueryRowContext(ctx, query, string(id))

	// Get the result.
	var count uint32
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// DuplicateSubjectExists determines whether or not a subject with the same external subject ID and a different
// internal subject ID exists.
func DuplicateSubjectExists(
	ctx context.Context,
	tx *sql.Tx,
	id models.InternalSubjectID,
	subjectID models.ExternalSubjectID,
) (bool, error) {

	// Query the database.
	query := "SELECT count(*) FROM subjects WHERE id != $1 and subject_id = $2"
	row := tx.QueryRowContext(ctx, query, string(id), string(subjectID))

	// Get the result.
	var count uint32
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListSubjects lists subjects in the database, optionally filtering by subject type and external subject ID
func ListSubjects(ctx context.Context, tx *sql.Tx, subjectType, subjectID *string) ([]*models.SubjectOut, error) {

	// Query the database.
	var rows *sql.Rows
	var err error
	if subjectType != nil && subjectID != nil {
		query := `SELECT id, subject_id, subject_type FROM subjects
		          WHERE subject_type = $1 AND subject_id = $2
		          ORDER BY subject_type, subject_id`
		rows, err = tx.QueryContext(ctx, query, *subjectType, *subjectID)
	} else if subjectType != nil {
		query := `SELECT id, subject_id, subject_type FROM subjects
		          WHERE subject_type = $1
		          ORDER BY subject_type, subject_id`
		rows, err = tx.QueryContext(ctx, query, *subjectType)
	} else if subjectID != nil {
		query := `SELECT id, subject_id, subject_type FROM subjects
		          WHERE subject_id = $1
		          ORDER BY subject_type, subject_id`
		rows, err = tx.QueryContext(ctx, query, *subjectID)
	} else {
		query := `SELECT id, subject_id, subject_type FROM subjects
		          ORDER BY subject_type, subject_id`
		rows, err = tx.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "subjects rows")

	// Get the list of subjects.
	return rowsToSubjectList(rows)
}

// DeleteSubject removes a subject from the database.
func DeleteSubject(ctx context.Context, tx *sql.Tx, id models.InternalSubjectID) error {

	// Update the database.
	stmt := "DELETE FROM subjects WHERE id = $1"
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
		return fmt.Errorf("no subjects deleted for id %s", id)
	}
	if count > 1 {
		return fmt.Errorf("multiple subjects deleted for id %s", id)
	}

	return nil
}

// GetSubject obtains information about the subject with the given external ID and subject type.
func GetSubject(
	ctx context.Context,
	tx *sql.Tx,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) (*models.SubjectOut, error) {

	// Get the subject information from the database.
	query := `SELECT id, subject_id, subject_type FROM subjects
            WHERE subject_id = $1 AND subject_type = $2`
	rows, err := tx.QueryContext(ctx, query, string(subjectID), string(subjectType))
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "subjects rows")

	// Get the subject.
	duplicateErr := fmt.Errorf(
		"found multiple subjects with ID, %s, and type, %s", string(subjectID), string(subjectType),
	)
	return rowsToSubject(rows, duplicateErr)
}

// GetSubjectByExternalID returns information about the subjects with the given external ID.
func GetSubjectByExternalID(ctx context.Context, tx *sql.Tx, subjectID models.ExternalSubjectID) (*models.SubjectOut, error) {

	// Get the subject information from the database.
	query := "SELECT id, subject_id, subject_type FROM subjects WHERE subject_id = $1"
	rows, err := tx.QueryContext(ctx, query, string(subjectID))
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "subjects rows")

	// Get the subject.
	duplicateErr := fmt.Errorf("found multiple subjects with ID, %s", string(subjectID))
	return rowsToSubject(rows, duplicateErr)
}
