package conf

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
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

// fakeStore implements SecretStore via a single function pointer so each test
// can wire up its own GetSecret behaviour.
type fakeStore struct {
	getSecret func(ctx context.Context, store, key string, meta map[string]string) (map[string]string, error)
}

func (f *fakeStore) GetSecret(ctx context.Context, store, key string, meta map[string]string) (map[string]string, error) {
	return f.getSecret(ctx, store, key, meta)
}

func discardHelper() *log.Helper {
	return log.NewHelper(log.NewStdLogger(io.Discard))
}

// withFastRetries shrinks the package-level retry knobs so tests don't sleep
// for real, restoring them on cleanup.
func withFastRetries(t *testing.T, attempts int, interval time.Duration) {
	t.Helper()
	prevAttempts, prevInterval := secretLoadAttempts, secretLoadInterval
	secretLoadAttempts, secretLoadInterval = attempts, interval
	t.Cleanup(func() {
		secretLoadAttempts, secretLoadInterval = prevAttempts, prevInterval
	})
}

func TestLoadSecrets_Success(t *testing.T) {
	// Arrange
	withFastRetries(t, 3, time.Millisecond)
	client := &fakeStore{
		getSecret: func(ctx context.Context, store, key string, _ map[string]string) (map[string]string, error) {
			assert.Equal(t, secretStoreName, store)
			assert.Equal(t, secretBundle, key)
			return map[string]string{
				keyDBSource:  "postgres://user:pass@host/db",
				keyRedisHost: "redis:6379",
			}, nil
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), client, discardHelper())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, &Secrets{
		DatabaseSource: "postgres://user:pass@host/db",
		RedisHost:      "redis:6379",
	}, secrets)
}

func TestLoadSecrets_MissingRequiredField(t *testing.T) {
	cases := []struct {
		name    string
		bundle  map[string]string
		missing string
	}{
		{
			name:    "missing database source",
			bundle:  map[string]string{keyRedisHost: "redis:6379"},
			missing: keyDBSource,
		},
		{
			name:    "missing redis host",
			bundle:  map[string]string{keyDBSource: "postgres://x"},
			missing: keyRedisHost,
		},
		{
			name: "empty database source",
			bundle: map[string]string{
				keyDBSource:  "",
				keyRedisHost: "redis:6379",
			},
			missing: keyDBSource,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			withFastRetries(t, 3, time.Millisecond)
			client := &fakeStore{
				getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
					return tc.bundle, nil
				},
			}

			// Act
			secrets, err := LoadSecrets(context.Background(), client, discardHelper())

			// Assert
			require.Error(t, err)
			assert.Nil(t, secrets)
			assert.Contains(t, err.Error(), tc.missing)
		})
	}
}

func TestLoadSecrets_RetriesUntilSuccess(t *testing.T) {
	// Arrange
	withFastRetries(t, 5, time.Millisecond)
	var calls atomic.Int32
	client := &fakeStore{
		getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
			if calls.Add(1) < 3 {
				return nil, errors.New("sidecar warming up")
			}
			return map[string]string{
				keyDBSource:  "postgres://x",
				keyRedisHost: "redis:6379",
			}, nil
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), client, discardHelper())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, secrets)
	assert.Equal(t, int32(3), calls.Load(), "expected exactly 3 attempts (2 failures + 1 success)")
}

func TestLoadSecrets_RetriesExhausted(t *testing.T) {
	// Arrange
	withFastRetries(t, 3, time.Millisecond)
	transientErr := errors.New("sidecar unreachable")
	var calls atomic.Int32
	client := &fakeStore{
		getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
			calls.Add(1)
			return nil, transientErr
		},
	}

	// Act
	secrets, err := LoadSecrets(context.Background(), client, discardHelper())

	// Assert
	require.Error(t, err)
	assert.Nil(t, secrets)
	assert.ErrorIs(t, err, transientErr)
	assert.Equal(t, int32(3), calls.Load(), "expected attempts to equal secretLoadAttempts")
}

func TestLoadSecrets_ContextCanceled(t *testing.T) {
	// Arrange — large attempt count + slow interval so retry loop must wait on ctx
	withFastRetries(t, 100, time.Second)
	client := &fakeStore{
		getSecret: func(context.Context, string, string, map[string]string) (map[string]string, error) {
			return nil, errors.New("transient")
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	secrets, err := LoadSecrets(ctx, client, discardHelper())

	// Assert
	require.Error(t, err)
	assert.Nil(t, secrets)
	assert.ErrorIs(t, err, context.Canceled)
}
