package conf

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-kratos/kratos/v2/log"
)

// SecretStore is the minimal interface LoadSecrets needs from a secret backend.
// Keeping the dependency one method wide makes the function trivial to fake.
type SecretStore interface {
	GetSecret(ctx context.Context, storeName, key string, meta map[string]string) (map[string]string, error)
}

const (
	secretStoreName = "secretstore"
	secretBundle    = "secrets"
	secretTagKey    = "secret"
)

// Retry knobs are vars (not consts) so tests can shrink them.
var (
	secretLoadAttempts = 12
	secretLoadInterval = 5 * time.Second
	secretCallTimeout  = 5 * time.Second
)

// Secrets holds runtime values pulled from the secret store. Each field's
// `secret:"..."` tag is the bundle key the value is read from; LoadSecrets
// requires every tagged field to have a non-empty value.
type Secrets struct {
	DatabaseSource string `secret:"DATABASE_CONNECTION_STRING"`
	RedisHost      string `secret:"REDIS_HOST"`
}

// LoadSecrets retries the secret store until it responds, then maps the bundle
// into a Secrets struct and verifies every tagged field is set.
// The caller owns the store lifecycle.
func LoadSecrets(ctx context.Context, store SecretStore, logger *log.Helper) (*Secrets, error) {
	var (
		raw     map[string]string
		attempt int
	)
	op := func() error {
		attempt++
		callCtx, cancel := context.WithTimeout(ctx, secretCallTimeout)
		defer cancel()
		s, err := store.GetSecret(callCtx, secretStoreName, secretBundle, nil)
		if err != nil {
			return err
		}
		raw = s
		return nil
	}
	notify := func(err error, _ time.Duration) {
		logger.Infof("waiting for secret store (attempt %d/%d): %v", attempt, secretLoadAttempts, err)
	}
	bo := backoff.WithContext(
		backoff.WithMaxRetries(backoff.NewConstantBackOff(secretLoadInterval), uint64(secretLoadAttempts-1)),
		ctx,
	)
	if err := backoff.RetryNotify(op, bo, notify); err != nil {
		return nil, fmt.Errorf("secret store not ready after %s: %w", time.Duration(secretLoadAttempts)*secretLoadInterval, err)
	}
	return mapSecrets(raw)
}

// mapSecrets copies tagged fields from raw into a Secrets struct, returning an
// error if any tagged field is missing or empty.
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
