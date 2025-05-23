package sqlite

import (
	"fmt"

	"github.com/goravel/framework/contracts/testing/docker"
	"github.com/goravel/framework/support/file"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	gormio "gorm.io/gorm"
)

type Docker struct {
	database string
}

func NewDocker(database string) *Docker {
	return &Docker{
		database: database,
	}
}

func (r *Docker) Build() error {
	if _, err := r.connect(); err != nil {
		return fmt.Errorf("connect Sqlite error: %v", err)
	}

	return nil
}

func (r *Docker) Config() docker.DatabaseConfig {
	return docker.DatabaseConfig{
		Database: r.database,
		Driver:   Name,
	}
}

func (r *Docker) Database(name string) (docker.DatabaseDriver, error) {
	docker := NewDocker(name)
	if err := docker.Build(); err != nil {
		return nil, err
	}

	return docker, nil
}

func (r *Docker) Driver() string {
	return Name
}

func (r *Docker) Fresh() error {
	if err := r.Shutdown(); err != nil {
		return err
	}

	if _, err := r.connect(); err != nil {
		return fmt.Errorf("connect Sqlite error when freshing: %v", err)
	}

	return nil
}

func (r *Docker) Image(image docker.Image) {
}

func (r *Docker) Ready() error {
	_, err := r.connect()

	return err
}

func (r *Docker) Reuse(containerID string, port int) error {
	return nil
}

func (r *Docker) Shutdown() error {
	if err := file.Remove(r.database); err != nil {
		return fmt.Errorf("stop Sqlite error: %v", err)
	}

	return nil
}

func (r *Docker) connect() (*gormio.DB, error) {
	return gormio.Open(gormlite.Open(r.database))
}
