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
	packages.Setup(os.Args).
		Install(
			modify.GoFile(path.Config("app.go")).
				Find(match.Imports()).Modify(modify.AddImport(packages.GetModulePath())).
				Find(match.Providers()).Modify(modify.Register("&sqlite.ServiceProvider{}", "&queue.ServiceProvider{}")),
			modify.GoFile(path.Config("database.go")).
				Find(match.Imports()).Modify(modify.AddImport("github.com/goravel/framework/contracts/database/driver"), modify.AddImport("github.com/goravel/sqlite/facades", "sqlitefacades")).
				Find(match.Config("database.connections")).Modify(modify.AddConfig("sqlite", config)),
		).
		Uninstall(
			modify.GoFile(path.Config("app.go")).
				Find(match.Providers()).Modify(modify.Unregister("&sqlite.ServiceProvider{}")).
				Find(match.Imports()).Modify(modify.RemoveImport(packages.GetModulePath())),
			modify.GoFile(path.Config("database.go")).
				Find(match.Config("database.connections")).Modify(modify.RemoveConfig("sqlite")).
				Find(match.Imports()).Modify(modify.RemoveImport("github.com/goravel/framework/contracts/database/driver"), modify.RemoveImport("github.com/goravel/sqlite/facades", "sqlitefacades")),
		).
		Execute()
}
