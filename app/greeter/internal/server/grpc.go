package server

import (
	coffeev1 "greeter/api/greeter/coffee/v1"
	v1 "greeter/api/greeter/helloworld/v1"
	"greeter/app/greeter/internal/conf"
	"greeter/app/greeter/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
// Note: kratos transport/grpc registers grpc.health.v1.Health by default —
// no explicit registration needed.
func NewGRPCServer(c *conf.Server, greeter *service.GreeterService, coffee *service.CoffeeService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			logging.Server(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGreeterServer(srv, greeter)
	coffeev1.RegisterCoffeeServer(srv, coffee)
	return srv
}
