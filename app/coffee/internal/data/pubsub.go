package data

import (
	"context"
	"encoding/json"
	"fmt"

	dapr "github.com/dapr/go-sdk/client"

	biz "coffee/app/coffee/internal/biz"
)

type daprPublisher struct {
	client dapr.Client
}

// NewPublisher returns a biz.Publisher backed by the Dapr pub/sub API.
func NewPublisher(daprClient dapr.Client) biz.Publisher {
	return &daprPublisher{client: daprClient}
}

func (p *daprPublisher) Publish(ctx context.Context, pubsubName, topic string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal pubsub payload: %w", err)
	}
	return p.client.PublishEvent(ctx, pubsubName, topic, payload, dapr.PublishEventWithContentType("application/json"))
}
