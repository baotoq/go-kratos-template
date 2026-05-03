package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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
	helper := log.NewHelper(logger)

	cfg := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer cfg.Close()

	if err := cfg.Load(); err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	daprClient, err := dapr.NewClient()
	if err != nil {
		panic(fmt.Errorf("dapr client: %w", err))
	}
	defer daprClient.Close()

	secrets, err := conf.LoadSecrets(context.Background(), daprClient, helper)
	if err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := cfg.Scan(&bc); err != nil {
		panic(fmt.Errorf("scan config: %w", err))
	}
	if bc.Server == nil || bc.Data == nil {
		panic("config: server and data sections are required")
	}
	bc.Data.Database.Source = secrets.DatabaseSource
	bc.Data.Redis.Addr = secrets.RedisHost

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(fmt.Errorf("wire app: %w", err))
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(fmt.Errorf("app run: %w", err))
	}
}
