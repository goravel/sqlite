package sqlite

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/goravel/framework/contracts/log"
	databasegorm "github.com/goravel/framework/database/gorm"
	"github.com/goravel/framework/support/carbon"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"

	"github.com/goravel/sqlite/contracts"
)

type Gorm struct {
	config contracts.ConfigBuilder
	log    log.Log
}

func NewGorm(configBuilder contracts.ConfigBuilder, log log.Log) *Gorm {
	return &Gorm{config: configBuilder, log: log}
}

func (r *Gorm) Build() (*gorm.DB, error) {
	instance, err := r.instance()
	if err != nil {
		return nil, err
	}
	if err := r.configurePool(instance); err != nil {
		return nil, err
	}
	if err := r.configureReadWriteSeparate(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

func (r *Gorm) configsToDialectors(configs []contracts.FullConfig) ([]gorm.Dialector, error) {
	var dialectors []gorm.Dialector

	for _, config := range configs {
		dsn := r.dns(config)
		if dsn == "" {
			return nil, FailedToGenerateDSN
		}

		dialector := sqlite.Open(dsn)
		dialectors = append(dialectors, dialector)
	}

	return dialectors, nil
}

func (r *Gorm) configurePool(instance *gorm.DB) error {
	db, err := instance.DB()
	if err != nil {
		return err
	}

	config := r.config.Config()
	db.SetMaxIdleConns(config.GetInt("database.pool.max_idle_conns", 10))
	db.SetMaxOpenConns(config.GetInt("database.pool.max_open_conns", 100))
	db.SetConnMaxIdleTime(time.Duration(config.GetInt("database.pool.conn_max_idletime", 3600)) * time.Second)
	db.SetConnMaxLifetime(time.Duration(config.GetInt("database.pool.conn_max_lifetime", 3600)) * time.Second)

	return nil
}

func (r *Gorm) configureReadWriteSeparate(instance *gorm.DB) error {
	writers, readers, err := r.writeAndReadDialectors()
	if err != nil {
		return err
	}

	return instance.Use(dbresolver.Register(dbresolver.Config{
		Sources:           writers,
		Replicas:          readers,
		Policy:            dbresolver.RandomPolicy{},
		TraceResolverMode: true,
	}))
}

func (r *Gorm) dns(config contracts.FullConfig) string {
	if config.Dsn != "" {
		return config.Dsn
	}

	return fmt.Sprintf("%s?multi_stmts=true", config.Database)
}

func (r *Gorm) gormConfig() *gorm.Config {
	logger := databasegorm.NewLogger(r.config.Config(), r.log)
	writeConfigs := r.config.Writes()
	if len(writeConfigs) == 0 {
		return nil
	}

	return &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		Logger:                                   logger,
		NowFunc: func() time.Time {
			return carbon.Now().StdTime()
		},
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   writeConfigs[0].Prefix,
			SingularTable: writeConfigs[0].Singular,
			NoLowerCase:   writeConfigs[0].NoLowerCase,
			NameReplacer:  writeConfigs[0].NameReplacer,
		},
	}
}

func (r *Gorm) instance() (*gorm.DB, error) {
	writers, _, err := r.writeAndReadDialectors()
	if err != nil {
		return nil, err
	}
	if len(writers) == 0 {
		return nil, ConfigNotFound
	}

	instance, err := gorm.Open(writers[0], r.gormConfig())
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (r *Gorm) writeAndReadDialectors() (writers []gorm.Dialector, readers []gorm.Dialector, err error) {
	writeConfigs := r.config.Writes()
	readConfigs := r.config.Reads()

	writers, err = r.configsToDialectors(writeConfigs)
	if err != nil {
		return nil, nil, err
	}
	readers, err = r.configsToDialectors(readConfigs)
	if err != nil {
		return nil, nil, err
	}

	return writers, readers, nil
}
