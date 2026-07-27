package db

import (
	"testing"

	"github.com/cyverse-de/permissions/models"
)

// TestPermissionDTOSourceID covers deriving the subject source ID from the
// subject type rather than asking Grouper for it.
//
// The wire values are Grouper's: `ldap` for users and `g:gsa` for groups.
// Callers -- terrain among them -- distinguish the two by this field, so the
// strings are a contract even though Grouper no longer supplies them.
func TestPermissionDTOSourceID(t *testing.T) {
	tests := []struct {
		name        string
		subjectType models.SubjectType
		want        models.SubjectSourceID
	}{
		{name: "a user", subjectType: models.SubjectTypeUser, want: "ldap"},
		{name: "a group", subjectType: models.SubjectTypeGroup, want: "g:gsa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := models.PermissionID("p1")
			internalID := models.InternalSubjectID("s1")
			subjectID := models.ExternalSubjectID("subject")
			subjectType := tt.subjectType
			level := models.PermissionLevel("read")

			dto := &PermissionDTO{
				ID:                &id,
				InternalSubjectID: &internalID,
				SubjectID:         &subjectID,
				SubjectType:       &subjectType,
				ResourceID:        "r1",
				ResourceName:      "resource",
				ResourceType:      "app",
				PermissionLevel:   &level,
			}

			got := dto.ToPermission()
			if got.Subject.SubjectSourceID == nil {
				t.Fatal("expected a subject source ID to be derived from the subject type")
			}
			if *got.Subject.SubjectSourceID != tt.want {
				t.Errorf("source ID = %q, want %q", *got.Subject.SubjectSourceID, tt.want)
			}
		})
	}
}
