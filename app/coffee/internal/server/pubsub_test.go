package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	biz "coffee/app/coffee/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

// fakeOrderHandler captures invocations and returns a configurable error.
type fakeOrderHandler struct {
	called   bool
	received biz.CoffeeOrderPlacedEvent
	err      error
}

func (f *fakeOrderHandler) HandleOrderPlaced(_ context.Context, evt biz.CoffeeOrderPlacedEvent) error {
	f.called = true
	f.received = evt
	return f.err
}

func discardLogger() *log.Helper {
	return log.NewHelper(log.NewStdLogger(io.Discard))
}

func TestSubscribeDiscovery_returns_subscription_list(t *testing.T) {
	// Arrange
	handler := subscribeDiscoveryHandler()
	req := httptest.NewRequest(http.MethodGet, "/dapr/subscribe", nil)
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var subs []map[string]string
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&subs))
	assert.Len(t, subs, 1)
	assert.Equal(t, biz.PubSubName, subs[0]["pubsubname"])
	assert.Equal(t, biz.TopicCoffeeOrders, subs[0]["topic"])
	assert.Equal(t, routeCoffeeOrders, subs[0]["route"])
}

func TestHandleOrder_decodes_cloudevent_and_dispatches(t *testing.T) {
	// Arrange
	fake := &fakeOrderHandler{}
	handler := handleCoffeeOrder(fake, discardLogger())
	body := `{"id":"1","source":"test","type":"com.dapr.event.sent","topic":"coffee.orders","pubsubname":"pubsub","datacontenttype":"application/json","data":{"instance_id":"abc123","beans":"arabica","size":"large"}}`
	req := httptest.NewRequest(http.MethodPost, routeCoffeeOrders, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"SUCCESS"}`, rec.Body.String())
	assert.True(t, fake.called)
	assert.Equal(t, biz.CoffeeOrderPlacedEvent{InstanceID: "abc123", Beans: "arabica", Size: "large"}, fake.received)
}

func TestHandleOrder_returns_RETRY_when_handler_fails(t *testing.T) {
	// Arrange
	fake := &fakeOrderHandler{err: errors.New("downstream unavailable")}
	handler := handleCoffeeOrder(fake, discardLogger())
	body := `{"data":{"instance_id":"abc","beans":"arabica","size":"small"}}`
	req := httptest.NewRequest(http.MethodPost, routeCoffeeOrders, strings.NewReader(body))
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"RETRY"}`, rec.Body.String())
	assert.True(t, fake.called)
}

func TestHandleOrder_rejects_bad_payload(t *testing.T) {
	// Arrange
	fake := &fakeOrderHandler{}
	handler := handleCoffeeOrder(fake, discardLogger())
	req := httptest.NewRequest(http.MethodPost, routeCoffeeOrders, strings.NewReader("not json at all"))
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.False(t, fake.called)
}

func TestHandleOrder_rejects_missing_data_field(t *testing.T) {
	// Arrange
	fake := &fakeOrderHandler{}
	handler := handleCoffeeOrder(fake, discardLogger())
	req := httptest.NewRequest(http.MethodPost, routeCoffeeOrders, strings.NewReader(`{"id":"1"}`))
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.False(t, fake.called)
}

func TestHandleOrder_rejects_non_post(t *testing.T) {
	// Arrange
	fake := &fakeOrderHandler{}
	handler := handleCoffeeOrder(fake, discardLogger())
	req := httptest.NewRequest(http.MethodGet, routeCoffeeOrders, nil)
	rec := httptest.NewRecorder()

	// Act
	handler(rec, req)

	// Assert
	resp := rec.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	assert.Equal(t, http.MethodPost, resp.Header.Get("Allow"))
	assert.False(t, fake.called)
}
