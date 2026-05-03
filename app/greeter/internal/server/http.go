package server

import (
	nethttp "net/http"

	v1 "greeter/api/greeter/helloworld/v1"
	ordersv1 "greeter/api/greeter/orders/v1"
	"greeter/app/greeter/internal/conf"
	"greeter/app/greeter/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, orders *service.OrdersService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.HandleFunc("/healthz", healthz)
	v1.RegisterGreeterHTTPServer(srv, greeter)
	ordersv1.RegisterOrdersHTTPServer(srv, orders)
	return srv
}

func healthz(w nethttp.ResponseWriter, _ *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
