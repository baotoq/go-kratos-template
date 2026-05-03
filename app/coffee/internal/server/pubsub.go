package server

import (
	"context"
	"encoding/json"
	nethttp "net/http"

	biz "coffee/app/coffee/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
)

const routeCoffeeOrders = "/api/coffee/orders"

// orderHandler is the subset of *biz.PubSubUsecase the subscription handler depends on.
// Declared locally so tests can inject a failing handler without standing up the full usecase.
type orderHandler interface {
	HandleOrderPlaced(ctx context.Context, evt biz.CoffeeOrderPlacedEvent) error
}

// cloudEventEnvelope captures the `data` field of a Dapr CloudEvents-v1.0 envelope.
// Our publisher always sends application/json, so Dapr delivers `data` as a JSON object.
type cloudEventEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// RegisterCoffeeSubscriptions registers the Dapr programmatic-subscription routes on srv.
func RegisterCoffeeSubscriptions(srv *http.Server, uc *biz.PubSubUsecase, logger log.Logger) {
	l := log.NewHelper(log.With(logger, "module", "server/pubsub"))
	srv.HandleFunc("/dapr/subscribe", subscribeDiscoveryHandler())
	srv.HandleFunc(routeCoffeeOrders, handleCoffeeOrder(uc, l))
}

func subscribeDiscoveryHandler() nethttp.HandlerFunc {
	type subscription struct {
		PubSubName string `json:"pubsubname"`
		Topic      string `json:"topic"`
		Route      string `json:"route"`
	}
	subs := []subscription{
		{PubSubName: biz.PubSubName, Topic: biz.TopicCoffeeOrders, Route: routeCoffeeOrders},
	}
	payload, _ := json.Marshal(subs)

	return func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(nethttp.StatusOK)
		_, _ = w.Write(payload)
	}
}

func handleCoffeeOrder(uc orderHandler, l *log.Helper) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			w.Header().Set("Allow", nethttp.MethodPost)
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		var envelope cloudEventEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			l.Warnw("msg", "failed to decode cloud event envelope", "err", err)
			w.WriteHeader(nethttp.StatusBadRequest)
			return
		}
		if len(envelope.Data) == 0 {
			l.Warnw("msg", "cloud event envelope missing data field")
			w.WriteHeader(nethttp.StatusBadRequest)
			return
		}

		var evt biz.CoffeeOrderPlacedEvent
		if err := json.Unmarshal(envelope.Data, &evt); err != nil {
			l.Warnw("msg", "failed to decode coffee order event", "err", err)
			w.WriteHeader(nethttp.StatusBadRequest)
			return
		}

		l.Infow("msg", "received coffee.orders event", "instance_id", evt.InstanceID)

		// Dapr HTTP-subscriber contract: 4xx = DROP, so a transient handler error must
		// be surfaced as 200 + status:RETRY (or 5xx) to ask the broker to redeliver.
		if err := uc.HandleOrderPlaced(r.Context(), evt); err != nil {
			l.Warnw("msg", "HandleOrderPlaced error; asking Dapr to retry", "err", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(nethttp.StatusOK)
			_, _ = w.Write([]byte(`{"status":"RETRY"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(nethttp.StatusOK)
		_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
	}
}
