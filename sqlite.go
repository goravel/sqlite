package sqlite

import (
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

func (r *Sqlite) Docker() (docker.DatabaseDriver, error) {
	writers := r.config.Writers()
	if len(writers) == 0 {
		return nil, errors.DatabaseConfigNotFound
	}

	return NewDocker(writers[0].Database), nil
}

func (r *Sqlite) Grammar() driver.Grammar {
	return NewGrammar(r.log, r.config.Writers()[0].Prefix)
}

func (r *Sqlite) Pool() database.Pool {
	return database.Pool{
		Readers: r.fullConfigsToConfigs(r.config.Readers()),
		Writers: r.fullConfigsToConfigs(r.config.Writers()),
	}
}

func (r *Sqlite) Processor() driver.Processor {
	return NewProcessor()
}

func (r *Sqlite) fullConfigsToConfigs(fullConfigs []contracts.FullConfig) []database.Config {
	configs := make([]database.Config, len(fullConfigs))
	for i, fullConfig := range fullConfigs {
		configs[i] = database.Config{
			Connection:   fullConfig.Connection,
			Dsn:          fullConfig.Dsn,
			Database:     fullConfig.Database,
			Dialector:    fullConfigToDialector(fullConfig),
			Driver:       Name,
			NameReplacer: fullConfig.NameReplacer,
			NoLowerCase:  fullConfig.NoLowerCase,
			Prefix:       fullConfig.Prefix,
			Singular:     fullConfig.Singular,
		}
	}

	return configs
}

func dsn(fullConfig contracts.FullConfig) string {
	if fullConfig.Dsn != "" {
		return fullConfig.Dsn
	}

	return fmt.Sprintf("%s?multi_stmts=true", fullConfig.Database)
}

func fullConfigToDialector(fullConfig contracts.FullConfig) gorm.Dialector {
	dsn := dsn(fullConfig)
	if dsn == "" {
		return nil
	}

	return sqlite.Open(dsn)
}
