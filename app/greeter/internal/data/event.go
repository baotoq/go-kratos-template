package data

import (
	"context"
	"greeter/app/greeter/internal/biz"
	"greeter/app/greeter/internal/conf"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/go-kratos/kratos/v2/log"
)

type eventRepo struct {
	client dapr.Client
	c      *conf.Data
	log    *log.Helper
}

func NewEventRepo(client dapr.Client, c *conf.Data, logger log.Logger) biz.EventRepo {
	return &eventRepo{
		client: client,
		c:      c,
		log:    log.NewHelper(logger),
	}
}

func (r *eventRepo) Publish(ctx context.Context, topic string, data []byte) error {
	if topic == "" {
		topic = r.c.Pubsub.Topic
	}
	return r.client.PublishEvent(ctx, r.c.Pubsub.Name, topic, data)
}
