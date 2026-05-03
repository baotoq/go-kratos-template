package biz

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

type spyPublisher struct {
	pubsubName string
	topic      string
	data       any
	err        error
}

func (s *spyPublisher) Publish(_ context.Context, pubsubName, topic string, data any) error {
	s.pubsubName = pubsubName
	s.topic = topic
	s.data = data
	return s.err
}

func newTestPubSubUsecase(p Publisher) *PubSubUsecase {
	return NewPubSubUsecase(p, log.NewStdLogger(io.Discard))
}

func TestPubSubUsecase_PublishOrderPlaced_success(t *testing.T) {
	// Arrange
	spy := &spyPublisher{}
	uc := newTestPubSubUsecase(spy)
	evt := CoffeeOrderPlacedEvent{InstanceID: "abc-123", Beans: "arabica", Size: "large"}

	// Act
	err := uc.PublishOrderPlaced(context.Background(), evt)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, PubSubName, spy.pubsubName)
	assert.Equal(t, TopicCoffeeOrders, spy.topic)
	assert.Equal(t, evt, spy.data)
}

func TestPubSubUsecase_PublishOrderPlaced_propagates_error(t *testing.T) {
	// Arrange
	publishErr := errors.New("broker unavailable")
	spy := &spyPublisher{err: publishErr}
	uc := newTestPubSubUsecase(spy)
	evt := CoffeeOrderPlacedEvent{InstanceID: "abc-123", Beans: "arabica", Size: "small"}

	// Act
	err := uc.PublishOrderPlaced(context.Background(), evt)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, publishErr)
}

func TestPubSubUsecase_HandleOrderPlaced_logs_and_returns_nil(t *testing.T) {
	// Arrange
	spy := &spyPublisher{}
	uc := newTestPubSubUsecase(spy)
	evt := CoffeeOrderPlacedEvent{InstanceID: "xyz-456", Beans: "robusta", Size: "medium"}

	// Act
	err := uc.HandleOrderPlaced(context.Background(), evt)

	// Assert
	assert.NoError(t, err)
}
