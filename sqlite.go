package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/errors"
	"gorm.io/gorm"

	"github.com/goravel/sqlite/contracts"
)

var _ driver.Driver = &Sqlite{}

type Sqlite struct {
	config contracts.ConfigBuilder
	db     *gorm.DB
	log    log.Log
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
		Version:    r.version(),
	}
}

func (r *Sqlite) DB() (*sql.DB, error) {
	gormDB, _, err := r.Gorm()
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

func (r *Sqlite) Gorm() (*gorm.DB, driver.GormQuery, error) {
	if r.db != nil {
		return r.db, NewQuery(), nil
	}

	db, err := NewGorm(r.config, r.log).Build()
	if err != nil {
		return nil, nil, err
	}

	r.db = db

	return db, NewQuery(), nil
}

func (r *Sqlite) Grammar() contractsschema.Grammar {
	return NewGrammar(r.log, r.config.Writes()[0].Prefix)
}

func (r *Sqlite) Processor() contractsschema.Processor {
	return NewProcessor()
}

func (r *Sqlite) version() string {
	instance, _, err := r.Gorm()
	if err != nil {
		return ""
	}

	var version struct {
		Value string
	}
	if err := instance.Raw("SELECT sqlite_version() AS value;").Scan(&version).Error; err != nil {
		return fmt.Sprintf("UNKNOWN: %s", err)
	}

	return version.Value
}
