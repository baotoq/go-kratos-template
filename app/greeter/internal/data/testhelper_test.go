package data_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// startPostgres launches a throwaway Postgres container and returns a
// libpq-style connection string with sslmode=disable. The container is
// torn down via t.Cleanup; callers should not stop it themselves.
func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	c, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("greeter_test"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return connStr
}

// startRedis launches a throwaway Redis container and returns a connected
// *redis.Client. The container is torn down via t.Cleanup.
func startRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	c, err := tcredis.Run(ctx, "redis:7-alpine")
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	connStr, err := c.ConnectionString(ctx)
	require.NoError(t, err)

	opt, err := redis.ParseURL(connStr)
	require.NoError(t, err)

	client := redis.NewClient(opt)
	t.Cleanup(func() { _ = client.Close() })
	return client
}
