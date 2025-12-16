package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/env"
	"github.com/goravel/framework/support/path"
)

func main() {
	setup := packages.Setup(os.Args)
	config := `map[string]any{
        "database": config.Env("DB_DATABASE", "forge"),
        "prefix":   "",
        "singular": false,
        "via": func() (driver.Driver, error) {
            return sqlitefacades.Sqlite("sqlite")
        },
    }`

	appConfigPath := path.Config("app.go")
	databaseConfigPath := path.Config("database.go")
	moduleImport := setup.Paths().Module().Import()
	sqliteServiceProvider := "&sqlite.ServiceProvider{}"
	driverContract := "github.com/goravel/framework/contracts/database/driver"
	sqliteFacades := "github.com/goravel/sqlite/facades"
	databaseConnectionsConfig := match.Config("database.connections")
	databaseConfig := match.Config("database")

	setup.Install(
		// Add sqlite service provider to app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Imports()).Modify(modify.AddImport(moduleImport)).
			Find(match.Providers()).Modify(modify.Register(sqliteServiceProvider))),

		// Add sqlite service provider to providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.AddProviderApply(moduleImport, sqliteServiceProvider)),

		// Add sqlite connection config to database.go
		modify.GoFile(path.Config("database.go")).
			Find(match.Imports()).Modify(
			modify.AddImport(driverContract),
			modify.AddImport(sqliteFacades, "sqlitefacades"),
		).
			Find(databaseConnectionsConfig).Modify(modify.AddConfig("sqlite", config)).
			Find(databaseConfig).Modify(modify.AddConfig("default", `"sqlite"`)),
	).Uninstall(
		// Remove sqlite connection config from database.go
		modify.GoFile(databaseConfigPath).
			Find(databaseConfig).Modify(modify.AddConfig("default", `""`)).
			Find(databaseConnectionsConfig).Modify(modify.RemoveConfig("sqlite")).
			Find(match.Imports()).Modify(
			modify.RemoveImport(driverContract),
			modify.RemoveImport(sqliteFacades, "sqlitefacades"),
		),

		// Remove sqlite service provider from app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Providers()).Modify(modify.Unregister(sqliteServiceProvider)).
			Find(match.Imports()).Modify(modify.RemoveImport(moduleImport))),

		// Remove sqlite service provider from providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.RemoveProviderApply(moduleImport, sqliteServiceProvider)),
	).Execute()
}
