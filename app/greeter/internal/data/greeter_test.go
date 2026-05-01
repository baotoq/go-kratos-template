package data_test

import (
	"context"
	"testing"

	"greeter/app/greeter/internal/biz"
	"greeter/app/greeter/internal/conf"
	"greeter/app/greeter/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGreeterRepo_SaveAndFindByID is the canonical example for new services
// adapted from this template: it exercises the real ent ORM against a
// Postgres container, going through NewData + NewGreeterRepo exactly like
// production wiring.
func TestGreeterRepo_SaveAndFindByID(t *testing.T) {
	t.Parallel()

	// Arrange
	connStr := startPostgres(t)
	d, cleanup, err := data.NewData(
		&conf.Data{
			Database: &conf.Data_Database{Driver: "postgres", Source: connStr},
		},
		log.DefaultLogger,
	)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	repo := data.NewGreeterRepo(d, log.DefaultLogger)
	ctx := context.Background()

	// Act
	saved, saveErr := repo.Save(ctx, &biz.Greeter{Hello: "world"})
	require.NoError(t, saveErr)
	got, getErr := repo.FindByID(ctx, int64(saved.ID))

	// Assert
	assert.NoError(t, getErr)
	assert.NotZero(t, saved.ID)
	assert.Equal(t, "world", saved.Hello)
	if assert.NotNil(t, got) {
		assert.Equal(t, saved.ID, got.ID)
		assert.Equal(t, "world", got.Hello)
	}
}

// TestGreeterRepo_FindByID_NotFound documents the error path so consumers
// know what to expect when a row does not exist.
func TestGreeterRepo_FindByID_NotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	connStr := startPostgres(t)
	d, cleanup, err := data.NewData(
		&conf.Data{
			Database: &conf.Data_Database{Driver: "postgres", Source: connStr},
		},
		log.DefaultLogger,
	)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	repo := data.NewGreeterRepo(d, log.DefaultLogger)

	// Act
	got, err := repo.FindByID(context.Background(), 999_999)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, got)
}
