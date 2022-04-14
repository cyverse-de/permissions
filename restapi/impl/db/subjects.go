package db

import (
	"context"
	"database/sql"
	"fmt"

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
func AddSubject(
	ctx context.Context,
	tx *sql.Tx,
	subjectID models.ExternalSubjectID,
	subjectType models.SubjectType,
) (*models.SubjectOut, error) {

	// Update the database.
	query := `INSERT INTO subjects (subject_id, subject_type) VALUES ($1, $2)
            RETURNING id, subject_id, subject_type`
	row := tx.QueryRowContext(ctx, query, string(subjectID), string(subjectType))

	// Return the subject information.
	var subjectDto SubjectDTO
	if err := row.Scan(&subjectDto.ID, &subjectDto.SubjectID, &subjectDto.SubjectType); err != nil {
		return nil, err
	}
	return subjectDto.ToSubjectOut(), nil
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
	defer rows.Close()

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
	defer rows.Close()

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
	defer rows.Close()

	// Get the subject.
	duplicateErr := fmt.Errorf("found multiple subjects with ID, %s", string(subjectID))
	return rowsToSubject(rows, duplicateErr)
}
