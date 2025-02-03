# Sqlite

The Sqlite driver for facades.Orm() of Goravel.

## Version

| goravel/sqlite | goravel/framework |
|------------------|-------------------|
| v1.0.*          | v1.16.*           |

## Install

1. Add package

```
go get -u github.com/goravel/sqlite
```

2. Register service provider

```
// config/app.go
import "github.com/goravel/sqlite"

"providers": []foundation.ServiceProvider{
    ...
    &sqlite.ServiceProvider{},
}
```

3. Add Sqlite driver to `config/database.go` file

```
// config/database.go
import (
    "github.com/goravel/framework/contracts/database"
    "github.com/goravel/framework/contracts/database/orm"
    sqlitefacades "github.com/goravel/sqlite/facades"
)

"connections": map[string]any{
    ...
    "sqlite": map[string]any{
        "database": config.Env("DB_DATABASE", "forge"),
        "prefix":   "",
        "singular": false,
        "via": func() (orm.Driver, error) {
            return sqlitefacades.Sqlite("sqlite"), nil
        },
        // Optional
        "read": []database.Config{
            {Database: "forge"},
        },
        // Optional
        "write": []database.Config{
            {Database: "forge"},
        },
    },
}
```
