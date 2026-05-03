package conf

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SecretStore is the minimal interface LoadSecrets needs from a Dapr-style
// secret backend. dapr.Client satisfies it structurally.
type SecretStore interface {
	Wait(ctx context.Context, timeout time.Duration) error
	GetSecret(ctx context.Context, storeName, key string, meta map[string]string) (map[string]string, error)
}

const (
	secretStoreName = "secretstore"
	secretTagKey    = "secret"
)

// secretWaitTimeout is a var so tests can shrink it.
var secretWaitTimeout = 60 * time.Second

// Secrets holds runtime values pulled from the secret store. Each field's
// `secret:"..."` tag is the secret key the value is read from; LoadSecrets
// requires every tagged field to have a non-empty value.
type Secrets struct {
	DatabaseSource string `secret:"DATABASE_CONNECTION_STRING"`
	RedisHost      string `secret:"REDIS_HOST"`
}

// LoadSecrets blocks until the sidecar reports ready, then fetches each tagged
// secret with one GetSecret call per key and verifies the value is non-empty.
// The caller owns the store lifecycle.
func LoadSecrets(ctx context.Context, store SecretStore, logger *log.Helper) (*Secrets, error) {
	logger.Info("waiting for secret store sidecar")
	if err := store.Wait(ctx, secretWaitTimeout); err != nil {
		return nil, fmt.Errorf("secret store not ready after %s: %w", secretWaitTimeout, err)
	}
	return mapSecrets(func(key string) (string, error) {
		result, err := store.GetSecret(ctx, secretStoreName, key, nil)
		if err != nil {
			return "", err
		}
		return result[key], nil
	})
}

// mapSecrets walks the Secrets struct and fills each tagged field by calling
// fetch(key). It returns an error if any tagged field's value is empty or if
// fetch fails. Decoupling this from the store means the reflection + validation
// rules live in one place and stay trivially testable.
func mapSecrets(fetch func(key string) (string, error)) (*Secrets, error) {
	var s Secrets
	v := reflect.ValueOf(&s).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		key, ok := t.Field(i).Tag.Lookup(secretTagKey)
		if !ok {
			continue
		}
		value, err := fetch(key)
		if err != nil {
			return nil, fmt.Errorf("get secret %q: %w", key, err)
		}
		if value == "" {
			return nil, fmt.Errorf("required secret %q is empty in store %q", key, secretStoreName)
		}
		v.Field(i).SetString(value)
	}
	return &s, nil
}
