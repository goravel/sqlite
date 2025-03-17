package contracts

import (
	contractsconfig "github.com/goravel/framework/contracts/config"
)

type ConfigBuilder interface {
	Config() contractsconfig.Config
	Connection() string
	Readers() []FullConfig
	Writers() []FullConfig
}

// Replacer replacer interface like strings.Replacer
type Replacer interface {
	Replace(name string) string
}

// Config Used in config/database.go
type Config struct {
	Dsn      string
	Database string
}

// FullConfig Fill the default value for Config
type FullConfig struct {
	Config
	Connection   string
	Driver       string
	NameReplacer Replacer
	NoLowerCase  bool
	Prefix       string
	Singular     bool
}
