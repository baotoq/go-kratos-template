package data_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

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
