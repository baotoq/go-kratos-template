package data

import (
	"context"
	"time"

	"greeter/app/greeter/internal/conf"
	"greeter/app/greeter/internal/data/ent"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/lib/pq"
)

var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

const schemaMigrateTimeout = 30 * time.Second

type Data struct {
	db *ent.Client
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	client, err := ent.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), schemaMigrateTimeout)
	defer cancel()
	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, nil, err
	}
	cleanup := func() {
		if err := client.Close(); err != nil {
			log.NewHelper(logger).Error(err)
		}
	}
	return &Data{db: client}, cleanup, nil
}
