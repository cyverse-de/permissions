package restapi

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cyverse-de/configurate"
	"github.com/cyverse-de/dbutil"
	"github.com/cyverse-de/go-mod/otelutils"
	"github.com/cyverse-de/version"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	errors "github.com/go-openapi/errors"
	httpkit "github.com/go-openapi/runtime"
	middleware "github.com/go-openapi/runtime/middleware"
	swag "github.com/go-openapi/swag"
	"github.com/spf13/viper"

	"github.com/cyverse-de/permissions/clients/grouper"
	"github.com/cyverse-de/permissions/logger"
	"github.com/cyverse-de/permissions/restapi/operations"
	"github.com/cyverse-de/permissions/restapi/operations/permissions"
	"github.com/cyverse-de/permissions/restapi/operations/resource_types"
	"github.com/cyverse-de/permissions/restapi/operations/resources"
	"github.com/cyverse-de/permissions/restapi/operations/status"
	"github.com/cyverse-de/permissions/restapi/operations/subjects"

	permissions_impl "github.com/cyverse-de/permissions/restapi/impl/permissions"
	resources_impl "github.com/cyverse-de/permissions/restapi/impl/resources"
	resource_types_impl "github.com/cyverse-de/permissions/restapi/impl/resourcetypes"
	status_impl "github.com/cyverse-de/permissions/restapi/impl/status"
	subjects_impl "github.com/cyverse-de/permissions/restapi/impl/subjects"

	// The blank import for the database driver appears in this package because it's the highest level package for
	// this service that isn't automatically generated.
	_ "github.com/lib/pq"
)

// This file is safe to edit. Once it exists it will not be overwritten

// DefaultConfig contains the default permissions configuration.
const DefaultConfig = `
db:
  uri: "postgresql://de:notprod@dedb:5432/de?sslmode=disable"
  schema: "permissions"

grouperdb:
  uri: "postgresql://de:notprod@adedb:5432/grouper?sslmode=disable"
  folder_name_prefix: "iplant:de:docker-compose"
`

// Command line options that aren't managed by go-swagger.
var options struct {
	CfgPath     string `long:"config" default:"/etc/iplant/de/permissions.yaml" description:"The path to the config file"`
	ShowVersion bool   `short:"v" long:"version" description:"Print the app version and exit"`
}

// Register the command-line options.
func configureFlags(api *operations.PermissionsAPI) {
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		{
			ShortDescription: "Service Options",
			LongDescription:  "",
			Options:          &options,
		},
	}
}

// Validate the custom command-line options.
func validateOptions() error {
	if options.CfgPath == "" {
		return fmt.Errorf("--config must be set")
	}

	return nil
}

// The database connection.
var db *sql.DB
var grouperClient *grouper.Client
var schema string
var tracerCtxCancel func()
var tracerShutdown func()

// Initialize the service.
func initService() error {
	logger.Log.Info("Initializing permissions service")
	if options.ShowVersion {
		version.AppVersion()
		os.Exit(0)
	}

	var (
		err error
		cfg *viper.Viper
	)

	var tracerCtx, cancel = context.WithCancel(context.Background())
	tracerCtxCancel = cancel
	tracerShutdown = otelutils.TracerProviderFromEnv(tracerCtx, "permissions", func(e error) { log.Fatal(e) })

	if cfg, err = configurate.InitDefaults(options.CfgPath, DefaultConfig); err != nil {
		return err
	}

	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		return err
	}

	dburi := cfg.GetString("db.uri")
	db, err = connector.Connect("postgres", dburi)
	if err != nil {
		return err
	}

	schema = cfg.GetString("db.schema")

	grouperDburi := cfg.GetString("grouperdb.uri")
	grouperFolderNamePrefix := cfg.GetString("grouperdb.folder_name_prefix")
	grouperClient, err = grouper.NewGrouperClient(grouperDburi, grouperFolderNamePrefix)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	logger.Log.Info("Done initializing")
	return nil
}

// Clean up when the service exits.
func cleanup() {
	defer tracerCtxCancel()
	logger.Log.Info("Closing the database connection.")
	db.Close()
	tracerShutdown()
}

func configureAPI(api *operations.PermissionsAPI) http.Handler {
	if err := validateOptions(); err != nil {
		logger.Log.Fatal(err)
	}

	if err := initService(); err != nil {
		logger.Log.Fatal(err)
	}

	api.ServeError = errors.ServeError

	api.JSONConsumer = httpkit.JSONConsumer()

	api.JSONProducer = httpkit.JSONProducer()

	api.StatusGetHandler = status.GetHandlerFunc(status_impl.BuildStatusHandler(SwaggerJSON))

	api.ResourceTypesGetResourceTypesHandler = resource_types.GetResourceTypesHandlerFunc(
		resource_types_impl.BuildResourceTypesGetHandler(db, schema),
	)

	api.ResourceTypesDeleteResourceTypeByNameHandler = resource_types.DeleteResourceTypeByNameHandlerFunc(
		resource_types_impl.BuildDeleteResourceTypeByNameHandler(db, schema),
	)

	api.ResourceTypesPostResourceTypesHandler = resource_types.PostResourceTypesHandlerFunc(
		resource_types_impl.BuildResourceTypesPostHandler(db, schema),
	)

	api.ResourceTypesPutResourceTypesIDHandler = resource_types.PutResourceTypesIDHandlerFunc(
		resource_types_impl.BuildResourceTypesIDPutHandler(db, schema),
	)

	api.ResourceTypesDeleteResourceTypesIDHandler = resource_types.DeleteResourceTypesIDHandlerFunc(
		resource_types_impl.BuildResourceTypesIDDeleteHandler(db, schema),
	)

	api.ResourcesAddResourceHandler = resources.AddResourceHandlerFunc(
		resources_impl.BuildAddResourceHandler(db, schema),
	)

	api.ResourcesDeleteResourceByNameHandler = resources.DeleteResourceByNameHandlerFunc(
		resources_impl.BuildDeleteResourceByNameHandler(db, schema),
	)

	api.ResourcesListResourcesHandler = resources.ListResourcesHandlerFunc(
		resources_impl.BuildListResourcesHandler(db, schema),
	)

	api.ResourcesUpdateResourceHandler = resources.UpdateResourceHandlerFunc(
		resources_impl.BuildUpdateResourceHandler(db, schema),
	)

	api.ResourcesDeleteResourceHandler = resources.DeleteResourceHandlerFunc(
		resources_impl.BuildDeleteResourceHandler(db, schema),
	)

	api.SubjectsAddSubjectHandler = subjects.AddSubjectHandlerFunc(
		subjects_impl.BuildAddSubjectHandler(db, schema),
	)

	api.SubjectsDeleteSubjectByExternalIDHandler = subjects.DeleteSubjectByExternalIDHandlerFunc(
		subjects_impl.BuildDeleteSubjectByExternalIDHandler(db, schema),
	)

	api.SubjectsListSubjectsHandler = subjects.ListSubjectsHandlerFunc(
		subjects_impl.BuildListSubjectsHandler(db, schema),
	)

	api.SubjectsUpdateSubjectHandler = subjects.UpdateSubjectHandlerFunc(
		subjects_impl.BuildUpdateSubjectHandler(db, schema),
	)

	api.SubjectsDeleteSubjectHandler = subjects.DeleteSubjectHandlerFunc(
		subjects_impl.BuildDeleteSubjectHandler(db, schema),
	)

	api.PermissionsListPermissionsHandler = permissions.ListPermissionsHandlerFunc(
		permissions_impl.BuildListPermissionsHandler(db, grouperClient, schema),
	)

	api.PermissionsGrantPermissionHandler = permissions.GrantPermissionHandlerFunc(
		permissions_impl.BuildGrantPermissionHandler(db, grouperClient, schema),
	)

	api.PermissionsRevokePermissionHandler = permissions.RevokePermissionHandlerFunc(
		permissions_impl.BuildRevokePermissionHandler(db, schema),
	)

	api.PermissionsPutPermissionHandler = permissions.PutPermissionHandlerFunc(
		permissions_impl.BuildPutPermissionHandler(db, grouperClient, schema),
	)

	api.PermissionsCopyPermissionsHandler = permissions.CopyPermissionsHandlerFunc(
		permissions_impl.BuildCopyPermissionsHandler(db, schema),
	)

	api.PermissionsBySubjectHandler = permissions.BySubjectHandlerFunc(
		permissions_impl.BuildBySubjectHandler(db, grouperClient, schema),
	)

	api.PermissionsBySubjectAndResourceTypeHandler = permissions.BySubjectAndResourceTypeHandlerFunc(
		permissions_impl.BuildBySubjectAndResourceTypeHandler(db, grouperClient, schema),
	)

	api.PermissionsBySubjectAndResourceTypeAbbreviatedHandler =
		permissions.BySubjectAndResourceTypeAbbreviatedHandlerFunc(
			permissions_impl.BuildBySubjectAndResourceTypeAbbreviatedHandler(db, grouperClient, schema),
		)

	api.PermissionsBySubjectAndResourceHandler = permissions.BySubjectAndResourceHandlerFunc(
		permissions_impl.BuildBySubjectAndResourceHandler(db, grouperClient, schema),
	)

	api.PermissionsListResourcePermissionsHandler = permissions.ListResourcePermissionsHandlerFunc(
		permissions_impl.BuildListResourcePermissionsHandler(db, grouperClient, schema),
	)

	api.ServerShutdown = cleanup

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, "HTTP request")
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json
// document. So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return middleware.Redoc(middleware.RedocOpts{}, handler)
}
