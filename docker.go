package sqlite

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/goravel/framework/contracts/testing"
	"github.com/goravel/framework/support/file"
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

func (r *Docker) Config() testing.DatabaseConfig {
	return testing.DatabaseConfig{
		Database: r.database,
	}
}

func (r *Docker) Database(name string) (testing.DatabaseDriver, error) {
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

func (r *Docker) Image(image testing.Image) {
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
	return gormio.Open(sqlite.Open(fmt.Sprintf("%s?multi_stmts=true", r.database)))
}
