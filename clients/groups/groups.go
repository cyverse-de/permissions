// Package groups provides an implementation of the grouper.Grouper interface
// backed by the DE groups service (Keycloak) instead of the Grouper database.
package groups

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/cyverse-de/permissions/models"
)

// Source IDs preserved from the Grouper era: permissions API clients still
// compare subject_source_id against these values, so they are part of the wire
// contract even though Grouper itself is going away.
const (
	userSubjectSource  = models.SubjectSourceID("ldap")
	groupSubjectSource = models.SubjectSourceID("g:gsa")
)

// errNotFound marks a 404 from the groups service so callers can decide
// whether a missing entity is an error.
var errNotFound = errors.New("not found")

// Client is a groups-service-backed implementation of grouper.Grouper.
type Client struct {
	base *url.URL
	user string
	hc   *http.Client
}

// NewClient validates the settings and returns a groups service client. The
// user is the trusted service account sent to the groups service as the acting
// user; it must be listed in the groups service's admin.users setting or
// membership listings will be rejected with a 403.
func NewClient(baseURI, user string) (*Client, error) {
	if baseURI == "" {
		return nil, errors.New("the groups service base URI must be specified")
	}
	if user == "" {
		return nil, errors.New("the groups service user must be specified")
	}
	base, err := url.Parse(baseURI)
	if err != nil {
		return nil, fmt.Errorf("invalid groups service base URI: %w", err)
	}
	return &Client{
		base: base,
		user: user,
		hc:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// get performs a GET against the groups service and decodes the JSON response
// body into out. A 404 response returns errNotFound.
func (c *Client) get(ctx context.Context, out interface{}, pathSegments ...string) error {
	requestURL := c.base.JoinPath(pathSegments...)
	query := requestURL.Query()
	query.Set("user", c.user)
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("groups service returned status %d for %s: %s", resp.StatusCode, requestURL.Path, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// subjectGroupsResponse matches the groups service GET /subjects/{id}/groups body.
type subjectGroupsResponse struct {
	Groups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"groups"`
}

// GroupsForSubject returns the groups the subject belongs to. An unknown
// subject yields an empty list, matching the Grouper database behavior.
func (c *Client) GroupsForSubject(ctx context.Context, subjectID string) ([]*grouper.GroupInfo, error) {
	var body subjectGroupsResponse
	if err := c.get(ctx, &body, "subjects", subjectID, "groups"); err != nil {
		if errors.Is(err, errNotFound) {
			return []*grouper.GroupInfo{}, nil
		}
		return nil, err
	}
	groups := make([]*grouper.GroupInfo, 0, len(body.Groups))
	for _, group := range body.Groups {
		groups = append(groups, &grouper.GroupInfo{ID: group.ID, Name: group.Name})
	}
	return groups, nil
}

// membersResponse matches the groups service GET /groups/{id}/members body.
type membersResponse struct {
	Members []struct {
		ID string `json:"id"`
	} `json:"members"`
}

// ListUsersInGroup returns the users belonging to a group. Group membership is
// flat (groups never contain groups), so every member is a user and the legacy
// "ldap" source ID is reported to preserve the wire contract. A missing group
// yields an empty list, matching the Grouper database behavior for groups that
// were deleted after a permission was granted to them.
func (c *Client) ListUsersInGroup(
	ctx context.Context,
	groupID models.ExternalSubjectID,
) ([]*models.SubjectOut, error) {
	var body membersResponse
	if err := c.get(ctx, &body, "groups", string(groupID), "members"); err != nil {
		if errors.Is(err, errNotFound) {
			return []*models.SubjectOut{}, nil
		}
		return nil, err
	}
	members := make([]*models.SubjectOut, 0, len(body.Members))
	for _, member := range body.Members {
		id := models.ExternalSubjectID(member.ID)
		source := userSubjectSource
		members = append(members, &models.SubjectOut{SubjectID: &id, SubjectSourceID: &source})
	}
	return members, nil
}
