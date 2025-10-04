package main

import (
	"os"

	"github.com/goravel/framework/packages"
	"github.com/goravel/framework/packages/match"
	"github.com/goravel/framework/packages/modify"
	"github.com/goravel/framework/support/path"
)

var config = `map[string]any{
        "database": config.Env("DB_DATABASE", "forge"),
        "prefix":   "",
        "singular": false,
        "via": func() (driver.Driver, error) {
            return sqlitefacades.Sqlite("sqlite")
        },
    }`

func main() {
	appConfigPath := path.Config("app.go")
	databaseConfigPath := path.Config("database.go")
	modulePath := packages.GetModulePath()
	sqliteServiceProvider := "&sqlite.ServiceProvider{}"
	driverContract := "github.com/goravel/framework/contracts/database/driver"
	sqliteFacades := "github.com/goravel/sqlite/facades"

	packages.Setup(os.Args).
		Install(
			// Add sqlite service provider to app.go
			modify.GoFile(appConfigPath).
				Find(match.Imports()).Modify(modify.AddImport(modulePath)).
				Find(match.Providers()).Modify(modify.Register(sqliteServiceProvider)),

			// Add sqlite connection config to database.go
			modify.GoFile(path.Config("database.go")).
				Find(match.Imports()).Modify(
				modify.AddImport(driverContract),
				modify.AddImport(sqliteFacades, "sqlitefacades"),
			).
				Find(match.Config("database.connections")).Modify(modify.AddConfig("sqlite", config)).
				Find(match.Config("database")).Modify(modify.AddConfig("default", `"sqlite"`)),
		).
		Uninstall(
			// Remove sqlite connection config from database.go
			modify.GoFile(databaseConfigPath).
				Find(match.Config("database")).Modify(modify.AddConfig("default", `""`)).
				Find(match.Config("database.connections")).Modify(modify.RemoveConfig("sqlite")).
				Find(match.Imports()).Modify(
				modify.RemoveImport(driverContract),
				modify.RemoveImport(sqliteFacades, "sqlitefacades"),
			),

			// Remove sqlite service provider from app.go
			modify.GoFile(appConfigPath).
				Find(match.Providers()).Modify(modify.Unregister(sqliteServiceProvider)).
				Find(match.Imports()).Modify(modify.RemoveImport(modulePath)),
		).
		Execute()
}
