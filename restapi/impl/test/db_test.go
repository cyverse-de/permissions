package test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

const testMaxExpandedSubjects = 10000

const FakeID = "1A960034-969A-46A7-B6B5-3F1866258CAB"

// skipUnlessIntegration skips the calling test unless it was asked for
// explicitly. These tests need a database, so without the skip a plain
// `go test ./...` reports them as passing without having asserted anything.
func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("set RUN_INTEGRATION_TESTS to run")
	}
}

// dburi names the DE database rather than a standalone one: the fixtures reach
// outside the permissions schema for public.users and the group tables.
func dburi() string {
	uri := os.Getenv("DBURI")
	if uri != "" {
		return uri
	}
	return "postgres://de:notprod@dedb:5432/de?sslmode=disable"
}

func schema() string {
	dbschema := os.Getenv("DBSCHEMA")
	if dbschema != "" {
		return dbschema
	}
	return "permissions"
}

func truncateTables(db *sql.DB, schema string) error {

	// Truncate all tables.
	tables := []string{"permissions", "subjects", "resources", "resource_types"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s.%s", schema, table))
		if err != nil {
			return err
		}
	}

	return nil
}

func initdb(t *testing.T) (*sql.DB, string) {
	db, err := sql.Open("postgres", dburi())
	if err != nil {
		t.Error(err)
	}
	err = db.Ping()
	if err != nil {
		t.Error(err)
	}

	dbschema := schema()

	if err := truncateTables(db, dbschema); err != nil {
		t.Error(err)
	}

	return db, dbschema
}

// addGroupMembership records a group's membership in the database.
//
// Group expansion reads group_effective_members, so the memberships the lookup
// tests rely on have to exist as rows. The closure is built by the same SQL
// function the services use, so the fixture exercises the real maintenance path
// rather than a hand-written approximation of it.
func addGroupMembership(t *testing.T, db *sql.DB, schema, groupID string, usernames ...string) {
	t.Helper()

	var groupInternalID string
	err := db.QueryRow(fmt.Sprintf(`
		INSERT INTO %[1]s.subjects (subject_id, subject_type) VALUES ($1, 'group')
		ON CONFLICT (subject_id) DO UPDATE SET subject_type = %[1]s.subjects.subject_type
		RETURNING id`, schema), groupID).Scan(&groupInternalID)
	if err != nil {
		t.Fatalf("could not create the group subject %s: %s", groupID, err)
	}

	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO %s.groups (subject_id, group_type, name) VALUES ($1, 'system', $2)
		ON CONFLICT (subject_id) DO NOTHING`, schema), groupInternalID, groupID)
	if err != nil {
		t.Fatalf("could not create the group %s: %s", groupID, err)
	}

	for _, username := range usernames {
		var memberID string
		err := db.QueryRow(fmt.Sprintf(`
			INSERT INTO %[1]s.subjects (subject_id, subject_type) VALUES ($1, 'user')
			ON CONFLICT (subject_id) DO UPDATE SET subject_type = %[1]s.subjects.subject_type
			RETURNING id`, schema), username).Scan(&memberID)
		if err != nil {
			t.Fatalf("could not create the member subject %s: %s", username, err)
		}

		_, err = db.Exec(fmt.Sprintf(`
			INSERT INTO %s.group_memberships (group_id, member_id, member_type)
			VALUES ($1, $2, 'user') ON CONFLICT DO NOTHING`, schema), groupInternalID, memberID)
		if err != nil {
			t.Fatalf("could not add %s to %s: %s", username, groupID, err)
		}
	}

	if _, err := db.Exec(
		fmt.Sprintf(`SELECT %s.recompute_group_closure(ARRAY[$1]::uuid[])`, schema),
		groupInternalID,
	); err != nil {
		t.Fatalf("could not recompute the closure for %s: %s", groupID, err)
	}
}

// addDefaultGroupMemberships mirrors the membership the lookup tests assume:
// s2 belongs to g1id, and s1 belongs to all three groups.
func addDefaultGroupMemberships(t *testing.T, db *sql.DB, schema string) {
	t.Helper()
	addGroupMembership(t, db, schema, "g1id", "s1", "s2")
	addGroupMembership(t, db, schema, "g2id", "s1")
	addGroupMembership(t, db, schema, "g3id", "s1")
}
