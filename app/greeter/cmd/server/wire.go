//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"greeter/app/greeter/internal/biz"
	"greeter/app/greeter/internal/conf"
	"greeter/app/greeter/internal/data"
	"greeter/app/greeter/internal/server"
	"greeter/app/greeter/internal/service"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, dapr.Client, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
