package data_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisContainer_SetGet is a worked example for services that will add
// a Redis-backed repo on top of this template. The template ships with a
// `data.redis` config block but no repo yet; copy this pattern when you
// add one.
func TestRedisContainer_SetGet(t *testing.T) {
	t.Parallel()

	// Arrange
	client := startRedis(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "greeting", "Hello, Kratos!", 0).Err())

	// Act
	got, getErr := client.Get(ctx, "greeting").Result()
	_, missErr := client.Get(ctx, "missing").Result()

	// Assert
	assert.NoError(t, getErr)
	assert.Equal(t, "Hello, Kratos!", got)
	assert.ErrorIs(t, missErr, redis.Nil)
}
