package main

import (
	"context"
	"flag"
	"fmt"
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

const (
	daprAttempts        = 12
	daprAttemptInterval = 5 * time.Second
	daprCallTimeout     = 5 * time.Second
	secretStoreName     = "secretstore"
	secretBundle        = "secrets"
	secretDBSource      = "DATABASE_CONNECTION_STRING"
	secretRedisHost     = "REDIS_HOST"
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

// loadDaprSecrets retries the Dapr secret store until the sidecar is ready.
// On success, the caller owns the returned client and must Close it.
func loadDaprSecrets(logger *log.Helper) (dapr.Client, map[string]string, error) {
	var lastErr error
	for attempt := 1; attempt <= daprAttempts; attempt++ {
		client, err := dapr.NewClient()
		if err != nil {
			lastErr = err
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), daprCallTimeout)
			s, secErr := client.GetSecret(ctx, secretStoreName, secretBundle, nil)
			cancel()
			if secErr == nil {
				return client, s, nil
			}
			client.Close()
			lastErr = secErr
		}
		if attempt == daprAttempts {
			break
		}
		logger.Infof("waiting for dapr sidecar (attempt %d/%d): %v", attempt, daprAttempts, lastErr)
		time.Sleep(daprAttemptInterval)
	}
	return nil, nil, fmt.Errorf("dapr sidecar not ready after %s: %w",
		time.Duration(daprAttempts)*daprAttemptInterval, lastErr)
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

	daprClient, secret, err := loadDaprSecrets(helper)
	if err != nil {
		panic(err)
	}
	defer daprClient.Close()

	for _, key := range []string{secretDBSource, secretRedisHost} {
		if secret[key] == "" {
			panic(fmt.Errorf("required secret %q is empty in dapr secretstore %q/%q", key, secretStoreName, secretBundle))
		}
	}

	var bc conf.Bootstrap
	if err := cfg.Scan(&bc); err != nil {
		panic(fmt.Errorf("scan config: %w", err))
	}
	if bc.Server == nil || bc.Data == nil {
		panic("config: server and data sections are required")
	}
	bc.Data.Database.Source = secret[secretDBSource]
	bc.Data.Redis.Addr = secret[secretRedisHost]

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(fmt.Errorf("wire app: %w", err))
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(fmt.Errorf("app run: %w", err))
	}
}
