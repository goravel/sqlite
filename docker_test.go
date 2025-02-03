package sqlite

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DockerTestSuite struct {
	suite.Suite
	database string
	docker   *Docker
}

func TestDockerTestSuite(t *testing.T) {
	suite.Run(t, new(DockerTestSuite))
}

func (s *DockerTestSuite) SetupTest() {
	s.database = "goravel"
	s.docker = NewDocker(s.database)
}

func (s *DockerTestSuite) Test_Build_Config_AddData_Fresh_Shutdown() {
	s.Nil(s.docker.Build())

	instance, err := s.docker.connect()
	s.Nil(err)
	s.NotNil(instance)

	s.Equal(s.database, s.docker.Config().Database)

	res := instance.Exec(`
CREATE TABLE users (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  name varchar(255) NOT NULL
);
`)
	s.Nil(res.Error)

	res = instance.Exec(`
INSERT INTO users (name) VALUES ('goravel');
`)
	s.Nil(res.Error)
	s.Equal(int64(1), res.RowsAffected)

	var count int64
	res = instance.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' and name = 'users';").Scan(&count)
	s.Nil(res.Error)
	s.Equal(int64(1), count)

	s.Nil(s.docker.Fresh())

	instance, err = s.docker.connect()
	s.Nil(err)
	s.NotNil(instance)

	res = instance.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' and name = 'users';").Scan(&count)
	s.Nil(res.Error)
	s.Equal(int64(0), count)

	databaseDriver, err := s.docker.Database("another")
	s.NoError(err)
	s.NotNil(databaseDriver)

	s.Nil(s.docker.Shutdown())
}

func (s *DockerTestSuite) TestDatabase() {
	s.Nil(s.docker.Build())

	_, err := s.docker.connect()
	s.Nil(err)

	docker, err := s.docker.Database("another")
	s.Nil(err)
	s.NotNil(docker)

	dockerImpl := docker.(*Docker)
	_, err = dockerImpl.connect()
	s.Nil(err)

	s.Nil(s.docker.Shutdown())
}
