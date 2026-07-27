package grouper

import (
	"context"

	"github.com/cyverse-de/permissions/models"
)

// MockGrouperClient represents a mock Grouper client.
type MockGrouperClient struct {
	groups      map[string][]*GroupInfo
	memberships map[string][]*models.SubjectOut
}

// NewEmptyMockGrouperClient returns a mock grouper client with no group or membership information.
func NewEmptyMockGrouperClient() *MockGrouperClient {
	return NewMockGrouperClient(nil, nil)
}

// NewMockGrouperClient returns a new mock grouper client. The groups parameter is a map from subject ID to a list of
// groups the subject belongs to. The memberships parameter is a map from group ID to a list of subject IDs. If either
// parameter is nil, an empty map will be used.
func NewMockGrouperClient(
	groups map[string][]*GroupInfo,
	memberships map[string][]*models.SubjectOut,
) *MockGrouperClient {
	if groups == nil {
		groups = make(map[string][]*GroupInfo)
	}
	if memberships == nil {
		memberships = make(map[string][]*models.SubjectOut)
	}
	return &MockGrouperClient{
		groups:      groups,
		memberships: memberships,
	}
}

// IsGroupSource returns true if the given source ID refers to a group in Grouper.
func (gc *MockGrouperClient) IsGroupSource(sourceID models.SubjectSourceID) bool {
	return sourceID == groupSubjectSource
}

// GroupsForSubject returns a mock list of groups for a subject.
func (gc *MockGrouperClient) GroupsForSubject(_ context.Context, subjectID string) ([]*GroupInfo, error) {
	return gc.groups[subjectID], nil
}

// ListUsersInGroup obtains a list of group members from the memberships field in the mock Grouper client. No attempts
// are made to mock nested groups at this time.
func (gc *MockGrouperClient) ListUsersInGroup(
	_ context.Context,
	subjectID models.ExternalSubjectID,
) ([]*models.SubjectOut, error) {
	return gc.memberships[string(subjectID)], nil
}
