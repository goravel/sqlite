package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/errors"
	"gorm.io/gorm"

	"github.com/goravel/sqlite/contracts"
)

var _ driver.Driver = &Sqlite{}

type Sqlite struct {
	config  contracts.ConfigBuilder
	db      *gorm.DB
	log     log.Log
	version string
}

func NewSqlite(config config.Config, log log.Log, connection string) *Sqlite {
	return &Sqlite{
		config: NewConfig(config, connection),
		log:    log,
	}
}

func (r *Sqlite) Config() database.Config {
	writers := r.config.Writes()
	if len(writers) == 0 {
		return database.Config{}
	}

	return database.Config{
		Connection: writers[0].Connection,
		Dsn:        writers[0].Dsn,
		Database:   writers[0].Database,
		Driver:     Name,
		Prefix:     writers[0].Prefix,
		Version:    r.getVersion(),
	}
}

func (r *Sqlite) DB() (*sql.DB, error) {
	gormDB, err := r.Gorm()
	if err != nil {
		return nil, err
	}

	return gormDB.DB()
}

func (r *Sqlite) Docker() (docker.DatabaseDriver, error) {
	writers := r.config.Writes()
	if len(writers) == 0 {
		return nil, errors.DatabaseConfigNotFound
	}

	return NewDocker(writers[0].Database), nil
}

func (r *Sqlite) Explain(sql string, vars ...any) string {
	return sqlite.Open("").Explain(sql, vars...)
}

func (r *Sqlite) Gorm() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}

	db, err := NewGorm(r.config, r.log).Build()
	if err != nil {
		return nil, err
	}

	r.db = db

	return db, nil
}

func (r *Sqlite) Grammar() driver.Grammar {
	return NewGrammar(r.log, r.config.Writes()[0].Prefix)
}

func (r *Sqlite) Processor() driver.Processor {
	return NewProcessor()
}

func (r *Sqlite) getVersion() string {
	if r.version != "" {
		return r.version
	}

	instance, err := r.Gorm()
	if err != nil {
		return ""
	}

	var version struct {
		Value string
	}
	if err := instance.Raw("SELECT sqlite_version() AS value;").Scan(&version).Error; err != nil {
		r.version = fmt.Sprintf("UNKNOWN: %s", err)
	} else {
		r.version = version.Value
	}

	return r.version
}
