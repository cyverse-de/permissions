package groups

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cyverse-de/permissions/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientValidation(t *testing.T) {
	tests := []struct {
		name    string
		baseURI string
		user    string
		wantErr bool
	}{
		{"valid", "http://groups", "permissions-svc", false},
		{"missing base URI", "", "permissions-svc", true},
		{"missing user", "http://groups", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.baseURI, tt.user)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGroupsForSubject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subjects/alice/groups", r.URL.Path)
		assert.Equal(t, "permissions-svc", r.URL.Query().Get("user"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[{"id":"g1","name":"team-a"},{"id":"g2","name":"team-b"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "permissions-svc")
	require.NoError(t, err)

	groups, err := client.GroupsForSubject(context.Background(), "alice")
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "g1", groups[0].ID)
	assert.Equal(t, "team-a", groups[0].Name)
	assert.Equal(t, "g2", groups[1].ID)
	assert.Equal(t, "team-b", groups[1].Name)
}

func TestGroupsForSubjectUnknownSubject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "permissions-svc")
	require.NoError(t, err)

	// The Grouper DB implementation returns an empty list for unknown
	// subjects; a 404 from the groups service must behave the same way.
	groups, err := client.GroupsForSubject(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestGroupsForSubjectServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "permissions-svc")
	require.NoError(t, err)

	_, err = client.GroupsForSubject(context.Background(), "alice")
	assert.Error(t, err)
}

func TestListUsersInGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/groups/g1/members", r.URL.Path)
		assert.Equal(t, "permissions-svc", r.URL.Query().Get("user"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"members":[{"id":"alice","name":"alice","source_id":"ldap-provider-uuid"},{"id":"bob","name":"bob"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "permissions-svc")
	require.NoError(t, err)

	members, err := client.ListUsersInGroup(context.Background(), models.ExternalSubjectID("g1"))
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, models.ExternalSubjectID("alice"), *members[0].SubjectID)
	assert.Equal(t, models.ExternalSubjectID("bob"), *members[1].SubjectID)
	// Group membership is flat, so every member is a user, and the legacy
	// "ldap" source is reported regardless of the Keycloak source_id value.
	assert.Equal(t, models.SubjectSourceID("ldap"), *members[0].SubjectSourceID)
	assert.Equal(t, models.SubjectSourceID("ldap"), *members[1].SubjectSourceID)
}

func TestListUsersInGroupMissingGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "permissions-svc")
	require.NoError(t, err)

	// The Grouper DB implementation returns an empty list for a group it
	// cannot find (e.g. one deleted after a permission was granted).
	members, err := client.ListUsersInGroup(context.Background(), models.ExternalSubjectID("gone"))
	require.NoError(t, err)
	assert.Empty(t, members)
}
