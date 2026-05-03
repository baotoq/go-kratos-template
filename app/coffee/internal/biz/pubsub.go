package biz

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// PubSubName is the Dapr pubsub component name (matches deploy/k8s/base/infra/dapr/pubsub.yaml).
const PubSubName = "pubsub"

// TopicCoffeeOrders is the topic published when a coffee order is placed.
const TopicCoffeeOrders = "coffee.orders"

// CoffeeOrderPlacedEvent is the payload published to TopicCoffeeOrders when Brew() schedules a workflow.
type CoffeeOrderPlacedEvent struct {
	InstanceID string `json:"instance_id"`
	Beans      string `json:"beans"`
	Size       string `json:"size"`
}

// Publisher is the data-layer contract that knows how to publish to a Dapr pubsub component.
// data implements this; biz consumes it.
type Publisher interface {
	Publish(ctx context.Context, pubsubName, topic string, data any) error
}

// PubSubUsecase publishes coffee.orders events and handles incoming subscriptions.
type PubSubUsecase struct {
	publisher Publisher
	log       *log.Helper
}

// NewPubSubUsecase wires a Publisher + logger into the pubsub usecase.
func NewPubSubUsecase(p Publisher, logger log.Logger) *PubSubUsecase {
	return &PubSubUsecase{
		publisher: p,
		log:       log.NewHelper(log.With(logger, "module", "biz/pubsub")),
	}
}

// PublishOrderPlaced publishes a CoffeeOrderPlacedEvent to TopicCoffeeOrders.
func (uc *PubSubUsecase) PublishOrderPlaced(ctx context.Context, evt CoffeeOrderPlacedEvent) error {
	if err := uc.publisher.Publish(ctx, PubSubName, TopicCoffeeOrders, evt); err != nil {
		return fmt.Errorf("publish %s: %w", TopicCoffeeOrders, err)
	}
	uc.log.WithContext(ctx).Infow(
		"msg", "published coffee.orders event",
		"instance_id", evt.InstanceID,
		"beans", evt.Beans,
		"size", evt.Size,
	)
	return nil
}

// HandleOrderPlaced is invoked by the HTTP subscription endpoint when Dapr delivers a coffee.orders event.
// In this example it just logs — a real barista service would notify staff, kick off prep, etc.
func (uc *PubSubUsecase) HandleOrderPlaced(ctx context.Context, evt CoffeeOrderPlacedEvent) error {
	uc.log.WithContext(ctx).Infow(
		"msg", "barista received coffee order",
		"instance_id", evt.InstanceID,
		"beans", evt.Beans,
		"size", evt.Size,
	)
	return nil
}
