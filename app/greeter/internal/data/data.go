package data

import (
	"context"
	"greeter/app/greeter/internal/data/ent"
	"greeter/app/greeter/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/lib/pq"
)

var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewDaprClient)

type Data struct {
	db *ent.Client
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	client, err := ent.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Schema.Create(context.Background()); err != nil {
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
