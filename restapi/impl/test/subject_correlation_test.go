package test

import (
	"database/sql"
	"testing"

	"github.com/cyverse-de/permissions/models"
)

const testUserSuffix = "@iplantcollaborative.org"

// deUserID inserts a DE user and returns its identifier.
func deUserID(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	var id string
	err := db.QueryRow(
		`INSERT INTO public.users (username) VALUES ($1)
		 ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
		 RETURNING id`, username).Scan(&id)
	if err != nil {
		t.Fatalf("could not create the DE user: %s", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM public.users WHERE username LIKE 'test-%'`); err != nil {
			t.Errorf("could not clean up DE users: %s", err)
		}
	})
	return id
}

// subjectUserID reads the correlation recorded for a subject.
func subjectUserID(t *testing.T, db *sql.DB, schema, subjectID string) *string {
	t.Helper()
	var userID *string
	err := db.QueryRow(
		`SELECT user_id::text FROM `+schema+`.subjects WHERE subject_id = $1`, subjectID).Scan(&userID)
	if err != nil {
		t.Fatalf("could not read the subject: %s", err)
	}
	return userID
}

// TestAddSubjectCorrelatesDEUser covers populating subjects.user_id when this
// service creates a subject. The groups service already does it on its own write
// path; without this, subjects created by a permission grant stay uncorrelated
// and the column is unreliable exactly where it looks reliable.
func TestAddSubjectCorrelatesDEUser(t *testing.T) {
	skipUnlessIntegration(t)
	db, schema := initdb(t)

	wantID := deUserID(t, db, "test-known"+testUserSuffix)

	t.Run("a user with a DE account is correlated", func(t *testing.T) {
		subjectID := models.ExternalSubjectID("test-known")
		subjectType := models.SubjectType("user")
		addSubject(db, schema, subjectID, subjectType)

		got := subjectUserID(t, db, schema, "test-known")
		if got == nil {
			t.Fatal("expected the subject to be correlated with its DE user")
		}
		if *got != wantID {
			t.Errorf("correlated with %s, want %s", *got, wantID)
		}
	})

	t.Run("a user who has never logged in is still created", func(t *testing.T) {
		subjectID := models.ExternalSubjectID("test-nologin")
		subjectType := models.SubjectType("user")
		addSubject(db, schema, subjectID, subjectType)

		if got := subjectUserID(t, db, schema, "test-nologin"); got != nil {
			t.Errorf("expected no correlation, got %s", *got)
		}
	})

	t.Run("a group is never correlated", func(t *testing.T) {
		// subjects_user_id_is_user rejects a group row carrying a user_id, so
		// this would fail on insert rather than merely record nothing.
		subjectID := models.ExternalSubjectID("test-known-group")
		subjectType := models.SubjectType("group")
		addSubject(db, schema, subjectID, subjectType)

		if got := subjectUserID(t, db, schema, "test-known-group"); got != nil {
			t.Errorf("expected no correlation for a group, got %s", *got)
		}
	})
}

// TestUpdateSubjectRecorrelatesDEUser covers keeping subjects.user_id in step
// with the subject a row now describes. A stale correlation attributes the
// renamed subject's permissions to the user it used to be, and a stale one on a
// row that becomes a group violates subjects_user_id_is_user outright.
func TestUpdateSubjectRecorrelatesDEUser(t *testing.T) {
	skipUnlessIntegration(t)
	db, schema := initdb(t)

	aliceID := deUserID(t, db, "test-alice"+testUserSuffix)
	bobID := deUserID(t, db, "test-bob"+testUserSuffix)

	t.Run("a rename moves the correlation to the new user", func(t *testing.T) {
		subject := addSubject(db, schema, models.ExternalSubjectID("test-alice"), models.SubjectType("user"))
		if got := subjectUserID(t, db, schema, "test-alice"); got == nil || *got != aliceID {
			t.Fatalf("correlated with %v, want %s", got, aliceID)
		}

		updateSubject(db, schema, *subject.ID, models.ExternalSubjectID("test-bob"), models.SubjectType("user"))

		got := subjectUserID(t, db, schema, "test-bob")
		if got == nil {
			t.Fatal("expected the renamed subject to be correlated")
		}
		if *got != bobID {
			t.Errorf("correlated with %s, want %s", *got, bobID)
		}
	})

	t.Run("an unchanged subject ID keeps its correlation", func(t *testing.T) {
		subject := addSubject(db, schema, models.ExternalSubjectID("test-alice"), models.SubjectType("user"))

		updateSubject(db, schema, *subject.ID, models.ExternalSubjectID("test-alice"), models.SubjectType("user"))

		if got := subjectUserID(t, db, schema, "test-alice"); got == nil || *got != aliceID {
			t.Errorf("correlated with %v, want %s", got, aliceID)
		}
	})

	t.Run("becoming a group clears the correlation", func(t *testing.T) {
		deUserID(t, db, "test-becomes-group"+testUserSuffix)
		subject := addSubject(db, schema, models.ExternalSubjectID("test-becomes-group"), models.SubjectType("user"))
		if subjectUserID(t, db, schema, "test-becomes-group") == nil {
			t.Fatal("expected the subject to start out correlated")
		}

		updateSubject(db, schema, *subject.ID, models.ExternalSubjectID("test-becomes-group"), models.SubjectType("group"))

		if got := subjectUserID(t, db, schema, "test-becomes-group"); got != nil {
			t.Errorf("expected no correlation for a group, got %s", *got)
		}
	})
}

// TestSourceIDDerivedFromSubjectType covers the source ID end to end, through a
// handler and the real queries, for both subject types. Nothing else asserts the
// group case, so a regression there would otherwise be invisible.
func TestSourceIDDerivedFromSubjectType(t *testing.T) {
	skipUnlessIntegration(t)
	db, schema := initdb(t)
	addDefaultResourceTypes(db, schema, t)
	addDefaultPermissions(db, schema)

	perms := listResourcePermissions(db, schema, "analysis", "analysis1", false)

	seen := map[string]string{}
	for _, p := range perms.Permissions {
		if p.Subject.SubjectSourceID == nil {
			t.Fatalf("no source ID for subject %s", *p.Subject.SubjectID)
		}
		seen[string(*p.Subject.SubjectType)] = string(*p.Subject.SubjectSourceID)
	}

	if got := seen["user"]; got != "ldap" {
		t.Errorf("user source ID = %q, want \"ldap\"", got)
	}
	if got := seen["group"]; got != "g:gsa" {
		t.Errorf("group source ID = %q, want \"g:gsa\"", got)
	}
}
