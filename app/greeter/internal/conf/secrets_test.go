package conf

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tag values mirrored from Secrets — kept here so the test breaks loudly if a
// tag is renamed without updating callers.
const (
	keyDBSource  = "DATABASE_CONNECTION_STRING"
	keyRedisHost = "REDIS_HOST"
)

// fakeStore implements SecretStore via function pointers so each test can wire
// up its own Wait / GetSecret behaviour.
type fakeStore struct {
	wait      func(ctx context.Context, timeout time.Duration) error
	getSecret func(ctx context.Context, store, key string, meta map[string]string) (map[string]string, error)
}

func (f *fakeStore) Wait(ctx context.Context, timeout time.Duration) error {
	if f.wait == nil {
		return nil
	}
	return f.wait(ctx, timeout)
}

func (f *fakeStore) GetSecret(ctx context.Context, store, key string, meta map[string]string) (map[string]string, error) {
	return f.getSecret(ctx, store, key, meta)
}

func discardHelper() *log.Helper {
	return log.NewHelper(log.NewStdLogger(io.Discard))
}

// withFastWait shrinks the package-level wait timeout so tests don't sleep for
// real, restoring it on cleanup.
func withFastWait(t *testing.T, timeout time.Duration) {
	t.Helper()
	prev := secretWaitTimeout
	secretWaitTimeout = timeout
	t.Cleanup(func() { secretWaitTimeout = prev })
}

func TestLoadSecrets_Success(t *testing.T) {
	// Arrange
	withFastWait(t, time.Millisecond)
	var calls int
	store := &fakeStore{
		getSecret: func(_ context.Context, storeName, key string, _ map[string]string) (map[string]string, error) {
			calls++
			assert.Equal(t, secretStoreName, storeName)
			assert.Equal(t, secretBundle, key)
			return map[string]string{
				keyDBSource:  "postgres://user:pass@host/db",
				keyRedisHost: "redis:6379",
			}, nil
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), store, discardHelper())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "expected exactly one GetSecret call")
	assert.Equal(t, &Secrets{
		DatabaseSource: "postgres://user:pass@host/db",
		RedisHost:      "redis:6379",
	}, secrets)
}

func TestLoadSecrets_MissingRequiredField(t *testing.T) {
	cases := []struct {
		name   string
		bundle map[string]string
		empty  string
	}{
		{
			name:   "missing database source",
			bundle: map[string]string{keyRedisHost: "redis:6379"},
			empty:  keyDBSource,
		},
		{
			name:   "missing redis host",
			bundle: map[string]string{keyDBSource: "postgres://x"},
			empty:  keyRedisHost,
		},
		{
			name: "empty database source",
			bundle: map[string]string{
				keyDBSource:  "",
				keyRedisHost: "redis:6379",
			},
			empty: keyDBSource,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			withFastWait(t, time.Millisecond)
			store := &fakeStore{
				getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
					return tc.bundle, nil
				},
			}

			// Act
			secrets, err := LoadSecrets(context.Background(), store, discardHelper())

			// Assert
			require.Error(t, err)
			assert.Nil(t, secrets)
			assert.Contains(t, err.Error(), tc.empty)
		})
	}
}

func TestLoadSecrets_GetSecretFails(t *testing.T) {
	// Arrange
	withFastWait(t, time.Millisecond)
	backendErr := errors.New("permission denied")
	store := &fakeStore{
		getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
			return nil, backendErr
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), store, discardHelper())

	// Assert
	require.Error(t, err)
	assert.Nil(t, secrets)
	assert.ErrorIs(t, err, backendErr)
}

func TestLoadSecrets_WaitFails(t *testing.T) {
	// Arrange
	withFastWait(t, time.Millisecond)
	waitErr := errors.New("sidecar unreachable")
	store := &fakeStore{
		wait: func(context.Context, time.Duration) error {
			return waitErr
		},
		getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
			t.Fatal("GetSecret should not be called when Wait fails")
			return nil, nil
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), store, discardHelper())

	// Assert
	require.Error(t, err)
	assert.Nil(t, secrets)
	assert.ErrorIs(t, err, waitErr)
}

func TestLoadSecrets_ContextCanceled(t *testing.T) {
	// Arrange
	withFastWait(t, time.Second)
	store := &fakeStore{
		wait: func(ctx context.Context, _ time.Duration) error {
			return ctx.Err()
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	secrets, err := LoadSecrets(ctx, store, discardHelper())

	// Assert
	require.Error(t, err)
	assert.Nil(t, secrets)
	assert.ErrorIs(t, err, context.Canceled)
}
