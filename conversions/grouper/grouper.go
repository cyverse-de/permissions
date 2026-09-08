// Package grouper connects the conversion commands to the Grouper database,
// which the permissions service itself no longer reads.
package grouper

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/cyverse-de/permissions/restapi"
	"github.com/spf13/viper"
)

// DefaultConfig extends the service defaults with the Grouper database settings.
const DefaultConfig = restapi.DefaultConfig + `
grouperdb:
  uri: "postgresql://de:notprod@adedb:5432/grouper?sslmode=disable"
  folder_name_prefix: "iplant:de:docker-compose"
`

// Open connects to the Grouper database named by the configuration and returns
// the connection along with the configured folder name prefix.
func Open(cfg *viper.Viper) (*sql.DB, string, error) {
	uri := cfg.GetString("grouperdb.uri")
	if uri == "" {
		return nil, "", errors.New("grouperdb.uri is not set")
	}
	folderNamePrefix := cfg.GetString("grouperdb.folder_name_prefix")
	if folderNamePrefix == "" {
		return nil, "", errors.New("grouperdb.folder_name_prefix is not set")
	}

	db, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, "", err
	}
	if err := db.Ping(); err != nil {
		db.Close() // nolint:errcheck
		return nil, "", fmt.Errorf("connecting to the Grouper database: %w", err)
	}
	return db, folderNamePrefix, nil
}

// DEUsersGroupID returns the Grouper identifier of the de-users group.
func DEUsersGroupID(db *sql.DB, folderNamePrefix string) (string, error) {
	var groupID string
	err := db.QueryRow(
		"SELECT id FROM grouper_groups WHERE name = $1",
		fmt.Sprintf("%s:users:de-users", folderNamePrefix),
	).Scan(&groupID)
	if err != nil {
		return "", fmt.Errorf("looking up the de-users group: %w", err)
	}
	return groupID, nil
}
