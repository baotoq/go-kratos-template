package main

import (
	"context"
	"flag"
	"os"
	"time"

	"greeter/app/greeter/internal/conf"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	// Retry Dapr client connection — sidecar may not be ready immediately on pod start.
	var (
		daprClient dapr.Client
		secret     map[string]string
	)
	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c, err := dapr.NewClient()
		if err == nil {
			s, err := c.GetSecret(ctx, "secretstore", "secrets", nil)
			cancel()
			if err == nil {
				daprClient = c
				secret = s
				break
			}
			c.Close()
		} else {
			cancel()
		}
		if attempt >= 12 {
			panic("dapr sidecar not ready after 60s")
		}
		log.NewHelper(logger).Infof("waiting for dapr sidecar (attempt %d/12)...", attempt)
		time.Sleep(5 * time.Second)
	}
	defer daprClient.Close()

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	if v := secret["DATABASE_CONNECTION_STRING"]; v != "" {
		bc.Data.Database.Source = v
	}
	if v := secret["REDIS_HOST"]; v != "" {
		bc.Data.Redis.Addr = v
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
