/*
This conversion utility registers all existing tools in the DE database as public tools in the permissions database.
This utility uses the same config as the permissions service to connect to these databases
(the --config commandline argument is required),
and assumes that the permissions database already registered the de-users group.
*/
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/cyverse-de/permissions/conversions/grouper"
	"github.com/cyverse-de/permissions/logger"

	"github.com/cyverse-de/configurate"
	"github.com/cyverse-de/version"

	_ "github.com/lib/pq"
)

func determineDEDatabaseURI(dedburi, dburi, dedbname string) (string, error) {
	if dedburi != "" {
		return dedburi, nil
	}

	// Parse the permissions database URI.
	permsdburi, err := url.Parse(dburi)
	if err != nil {
		return "", err
	}

	// Create a new URI based on the permissions database URI.
	uri := &url.URL{
		Scheme:   permsdburi.Scheme,
		Opaque:   permsdburi.Opaque,
		User:     permsdburi.User,
		Host:     permsdburi.Host,
		Path:     fmt.Sprintf("/%s", dedbname),
		RawPath:  permsdburi.RawPath,
		RawQuery: permsdburi.RawQuery,
		Fragment: permsdburi.Fragment,
	}
	return uri.String(), nil
}

func listToolIDs(deDb *sql.DB) (*sql.Rows, error) {
	query := "SELECT id FROM tools"
	return deDb.Query(query)
}

func getResourceID(db *sql.DB, toolID string) (*string, error) {
	query := `SELECT id FROM resources
            WHERE resource_type_id = (SELECT id FROM resource_types WHERE name = 'tool')
            AND name = $1`
	rows, err := db.Query(query, toolID)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "resource ID rows")

	// Quit now if there are no matching rows.
	if !rows.Next() {
		return nil, nil
	}

	// Extract the resource ID from the first row.
	var resourceID *string
	if err := rows.Scan(&resourceID); err != nil {
		return nil, err
	}
	return resourceID, nil
}

func addResource(db *sql.DB, toolID string) (*string, error) {
	query := `INSERT INTO resources (name, resource_type_id)
            (SELECT $1, id FROM resource_types WHERE name = 'tool')
            RETURNING id`
	row := db.QueryRow(query, toolID)

	// Extract the resource ID.
	var resourceID *string
	if err := row.Scan(&resourceID); err != nil {
		logger.Log.Error(err)
		return nil, err
	}
	return resourceID, nil
}

func lookUpResourceID(db *sql.DB, toolID string) (*string, error) {
	resourceID, err := getResourceID(db, toolID)
	if err != nil {
		return nil, err
	}
	if resourceID != nil {
		return resourceID, nil
	}
	return addResource(db, toolID)
}

func registerTool(db *sql.DB, toolID, subjectID, level string) error {

	// Look up the tool ID, adding the tool to the database if necessary.
	resourceID, err := lookUpResourceID(db, toolID)
	if err != nil {
		return err
	}

	// Add the permission.
	stmt := `INSERT INTO permissions (subject_id, resource_id, permission_level_id)
           (SELECT $1, $2, id FROM permission_levels WHERE name = $3)`
	_, err = db.Exec(stmt, subjectID, *resourceID, level)
	return err
}

func runConversion(db, deDb *sql.DB, deUsersSubjectID string) error {

	// Get the tool listing from the DE database.
	tools, err := listToolIDs(deDb)
	if err != nil {
		return err
	}
	defer logger.LogClose(tools, "tools rows")

	// Register each tool in the permissions database (every tool is currently considered public).
	var toolID *string
	for tools.Next() {
		if err := tools.Scan(&toolID); err != nil {
			return err
		}
		if err := registerTool(db, *toolID, deUsersSubjectID, "read"); err != nil {
			return err
		}
	}

	return nil
}

func getSubjectID(db *sql.DB, subjectType, externalSubjectID string) (*string, error) {
	query := "SELECT id FROM subjects WHERE subject_type = $1 AND subject_id = $2"
	rows, err := db.Query(query, subjectType, externalSubjectID)
	if err != nil {
		return nil, err
	}
	defer logger.LogClose(rows, "subject ID rows")

	// Quit now if there are no matching rows.
	if !rows.Next() {
		return nil, nil
	}

	// Extract the subject ID from the first row.
	var subjectID *string
	if err := rows.Scan(&subjectID); err != nil {
		return nil, err
	}
	return subjectID, nil
}

func main() {
	config := flag.String("config", "", "The path to the configuration file.")
	deDburi := flag.String("de-database-uri", "", "The URI to use when connecting to the DE database.")
	deDbname := flag.String("de-database-name", "de", "The name of the DE database.")
	showVersion := flag.Bool("version", false, "Display version information and exit.")

	// Parse the command line arguments.
	flag.Parse()

	// Print the version information and exit if we're told to.
	if *showVersion {
		version.AppVersion()
		os.Exit(0)
	}

	// Validate the command-line options.
	if *config == "" {
		logger.Log.Fatal("--config must be set")
	}

	// Load the configuration file.
	cfg, err := configurate.InitDefaults(*config, grouper.DefaultConfig)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}

	// Establish the permissions database session.
	dburi := cfg.GetString("db.uri")
	db, err := sql.Open("postgres", dburi)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}
	defer logger.LogClose(db, "permissions database connection")
	if err := db.Ping(); err != nil {
		logger.Log.Fatal(err.Error())
	}

	// Establish the Grouper database session.
	grouperDb, grouperFolderNamePrefix, err := grouper.Open(cfg)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}
	defer logger.LogClose(grouperDb, "Grouper database connection")

	// Determine the DE Users group ID.
	deUsersGroupID, err := grouper.DEUsersGroupID(grouperDb, grouperFolderNamePrefix)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}

	// Look up the de-users subject ID.
	deUsersSubjectID, err := getSubjectID(db, "group", deUsersGroupID)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}
	if deUsersSubjectID == nil { // nolint:staticcheck
		logger.Log.Fatal("Could not find subject ID for de-users Group ID: ", deUsersGroupID)
	}

	// Determine the DE database URI.
	*deDburi, err = determineDEDatabaseURI(*deDburi, dburi, *deDbname)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}

	// Establish the connection to the DE database.
	deDb, err := sql.Open("postgres", *deDburi)
	if err != nil {
		logger.Log.Fatal(err.Error())
	}
	defer logger.LogClose(deDb, "DE database connection")
	if err := deDb.Ping(); err != nil {
		logger.Log.Fatal(err.Error())
	}

	// Run the conversion.
	if err := runConversion(db, deDb, *deUsersSubjectID); err != nil { // nolint:staticcheck
		logger.Log.Fatal(err.Error())
	}
}
