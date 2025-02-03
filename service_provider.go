package sqlite

import (
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/errors"
)

const (
	Binding = "goravel.sqlite"
	Name    = "sqlite"
)

var App foundation.Application

type ServiceProvider struct {
}

func (receiver *ServiceProvider) Register(app foundation.Application) {
	App = app

	app.BindWith(Binding, func(app foundation.Application, parameters map[string]any) (any, error) {
		config := app.MakeConfig()
		if config == nil {
			return nil, errors.ConfigFacadeNotSet.SetModule(Name)
		}

		log := app.MakeLog()
		if log == nil {
			return nil, errors.LogFacadeNotSet.SetModule(Name)
		}

		return NewSqlite(config, log, parameters["connection"].(string)), nil
	})
}

func (receiver *ServiceProvider) Boot(app foundation.Application) {

}
