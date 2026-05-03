package data_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisContainer_SetGet is a worked example for services that will add
// a Redis-backed repo on top of this template — the testcontainers + AAA
// pattern callers should mirror.
func TestRedisContainer_SetGet(t *testing.T) {
	t.Parallel()

	// Arrange
	client := startRedis(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "drink", "espresso", 0).Err())

	// Act
	got, getErr := client.Get(ctx, "drink").Result()
	_, missErr := client.Get(ctx, "missing").Result()

	// Assert
	assert.NoError(t, getErr)
	assert.Equal(t, "espresso", got)
	assert.ErrorIs(t, missErr, redis.Nil)
}
