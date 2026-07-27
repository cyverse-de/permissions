package grouper

import (
	"context"
	"database/sql"

	"github.com/cyverse-de/dbutil"
	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/models"
	"go.opentelemetry.io/otel"
)

const otelName = "github.com/cyverse-de/permissions/clients/grouper"
const groupSubjectSource = models.SubjectSourceID("g:gsa")

// GroupInfo represents information about a group in Grouper.
type GroupInfo struct {
	ID   string
	Name string
}

// Grouper is the interface implemented by a Grouper client instance.
type Grouper interface {
	IsGroupSource(sourceID models.SubjectSourceID) bool
	GroupsForSubject(context.Context, string) ([]*GroupInfo, error)
	ListUsersInGroup(context.Context, models.ExternalSubjectID) ([]*models.SubjectOut, error)
}

// Client represents a Grouper client instance.
//
// Note: the grouper client is intended to be a read-only client. Explicit transactions are not
// used here for that reason.
type Client struct {
	db     *sql.DB
	prefix string
}

// NewGrouperClient creates and returns a new Grouper client for the given database URI and group name prefix.
func NewGrouperClient(dbURI, prefix string) (*Client, error) {
	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		return nil, err
	}

	db, err := connector.Connect("postgres", dbURI)
	if err != nil {
		return nil, err
	}

	return &Client{
		db:     db,
		prefix: prefix,
	}, nil
}

// IsGroupSource returns true if the given source ID refers to a group in Grouper.
func (gc *Client) IsGroupSource(sourceID models.SubjectSourceID) bool {
	return sourceID == groupSubjectSource
}

// GroupsForSubject returns the list of groups that the subject with the given ID belongs to.
func (gc *Client) GroupsForSubject(ctx context.Context, subjectID string) ([]*GroupInfo, error) {
	ctx, span := otel.Tracer(otelName).Start(ctx, "GroupsForSubject")
	defer span.End()

	// Query the database.
	query := `SELECT group_id, group_name FROM grouper_memberships_v
            WHERE subject_id = $1 AND group_name LIKE $2 AND list_name = 'members'`
	rows, err := gc.db.QueryContext(ctx, query, subjectID, gc.prefix+"%")
	if err != nil {
		return nil, err
	}

	// Extract the groups from the database.
	groups := make([]*GroupInfo, 0)
	for rows.Next() {
		var group GroupInfo
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// ListUsersInGroup returns the list of users in the group with the given ID. Nested groups are supported, but only the
// users in the group or any nested groups are returned.
func (gc *Client) ListUsersInGroup(
	ctx context.Context,
	groupID models.ExternalSubjectID,
) ([]*models.SubjectOut, error) {
	ctx, span := otel.Tracer(otelName).Start(ctx, "ListUsersInGroup")
	defer span.End()

	// Query the database.
	query := `SELECT subject_id, subject_source FROM grouper_memberships_v
            WHERE group_id = $1 AND list_name = 'members' AND subject_source != $2`
	rows, err := gc.db.QueryContext(ctx, query, string(groupID), string(groupSubjectSource))
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "grouper memberships rows")

	// Build the list of subjects.
	members := make([]*models.SubjectOut, 0)
	for rows.Next() {
		var id models.ExternalSubjectID
		var source models.SubjectSourceID
		if err := rows.Scan(&id, &source); err != nil {
			return nil, err
		}
		members = append(members, &models.SubjectOut{SubjectID: &id, SubjectSourceID: &source})
	}

	return members, nil
}
