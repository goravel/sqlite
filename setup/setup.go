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
	driver := "sqlite"
	config := `map[string]any{
        "database": config.Env("DB_DATABASE", "forge"),
        "prefix":   "",
        "singular": false,
        "via": func() (driver.Driver, error) {
            return sqlitefacades.Sqlite("` + driver + `")
        },
    }`

	appConfigPath := path.Config("app.go")
	databaseConfigPath := path.Config("database.go")
	moduleImport := setup.Paths().Module().Import()
	sqliteServiceProvider := "&sqlite.ServiceProvider{}"
	driverContract := "github.com/goravel/framework/contracts/database/driver"
	sqliteFacades := "github.com/goravel/sqlite/facades"
	databaseConnectionsConfig := match.Config("database.connections")

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
		}, modify.RegisterProvider(moduleImport, sqliteServiceProvider)),

		// Add sqlite connection config to database.go
		modify.GoFile(path.Config("database.go")).
			Find(match.Imports()).Modify(
			modify.AddImport(driverContract),
			modify.AddImport(sqliteFacades, "sqlitefacades"),
		).Find(databaseConnectionsConfig).Modify(modify.AddConfig(driver, config)),

		// Add DB_CONNECTION=sqlite to .env
		modify.WhenFileExists(path.Base(".env"), modify.Env(path.Base(".env"), "DB_CONNECTION", driver)),
		modify.WhenFileExists(path.Base(".env.example"), modify.Env(path.Base(".env.example"), "DB_CONNECTION", driver)),
	).Uninstall(
		// Remove sqlite connection config from database.go
		modify.WhenFileExists(databaseConfigPath, modify.GoFile(databaseConfigPath).
			Find(databaseConnectionsConfig).Modify(modify.RemoveConfig(driver)).
			Find(match.Imports()).Modify(
			modify.RemoveImport(driverContract),
			modify.RemoveImport(sqliteFacades, "sqlitefacades"),
		)),

		// Remove sqlite service provider from app.go if not using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return !env.IsBootstrapSetup()
		}, modify.GoFile(appConfigPath).
			Find(match.Providers()).Modify(modify.Unregister(sqliteServiceProvider)).
			Find(match.Imports()).Modify(modify.RemoveImport(moduleImport))),

		// Remove sqlite service provider from providers.go if using bootstrap setup
		modify.When(func(_ map[string]any) bool {
			return env.IsBootstrapSetup()
		}, modify.UnregisterProvider(moduleImport, sqliteServiceProvider)),
	).Execute()
}
