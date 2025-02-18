package facades

import (
	"fmt"

	"github.com/goravel/framework/contracts/database/driver"

	"github.com/goravel/sqlite"
)

func Sqlite(connection string) (driver.Driver, error) {
	if sqlite.App == nil {

		return nil, fmt.Errorf("please register sqlite service provider")
	}

	instance, err := sqlite.App.MakeWith(sqlite.Binding, map[string]any{
		"connection": connection,
	})
	if err != nil {
		return nil, err
	}

	return instance.(*sqlite.Sqlite), nil
}
