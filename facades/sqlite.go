package facades

import (
	"log"

	"github.com/goravel/framework/contracts/database/driver"

	"github.com/goravel/sqlite"
)

func Sqlite(connection string) driver.Driver {
	if sqlite.App == nil {
		log.Fatalln("please register Sqlite service provider")
		return nil
	}

	instance, err := sqlite.App.MakeWith(sqlite.Binding, map[string]any{
		"connection": connection,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return instance.(*sqlite.Sqlite)
}
