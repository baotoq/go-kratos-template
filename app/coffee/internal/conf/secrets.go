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
	secretBundle    = "secrets"
	secretTagKey    = "secret"
)

// secretWaitTimeout is a var so tests can shrink it.
var secretWaitTimeout = 60 * time.Second

// Secrets holds runtime values pulled from the secret store. Each field's
// `secret:"..."` tag is the secret key the value is read from; LoadSecrets
// requires every tagged field to have a non-empty value.
type Secrets struct {
	RedisHost string `secret:"REDIS_HOST"`
}

// LoadSecrets blocks until the sidecar reports ready, then fetches the secret
// bundle in one GetSecret call and maps it onto a Secrets struct, verifying
// every tagged field is non-empty. The caller owns the store lifecycle.
func LoadSecrets(ctx context.Context, store SecretStore, logger *log.Helper) (*Secrets, error) {
	logger.Info("waiting for secret store to be ready")
	if err := store.Wait(ctx, secretWaitTimeout); err != nil {
		return nil, fmt.Errorf("secret store not ready after %s: %w", secretWaitTimeout, err)
	}
	raw, err := store.GetSecret(ctx, secretStoreName, secretBundle, nil)
	if err != nil {
		return nil, fmt.Errorf("get secret bundle %q: %w", secretBundle, err)
	}

	logger.Info("retrieved secrets")
	return mapSecrets(raw)
}

// mapSecrets walks the Secrets struct and copies each tagged field's value out
// of raw. It returns an error if any tagged field is missing or empty.
func mapSecrets(raw map[string]string) (*Secrets, error) {
	var s Secrets
	v := reflect.ValueOf(&s).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		key, ok := t.Field(i).Tag.Lookup(secretTagKey)
		if !ok {
			continue
		}
		value := raw[key]
		if value == "" {
			return nil, fmt.Errorf("required secret %q is empty in store %q bundle %q",
				key, secretStoreName, secretBundle)
		}
		v.Field(i).SetString(value)
	}
	return &s, nil
}
