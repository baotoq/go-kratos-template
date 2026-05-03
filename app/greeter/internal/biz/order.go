package biz

import (
	"context"
	"errors"
	"fmt"

	"github.com/dapr/durabletask-go/api"
	"github.com/dapr/durabletask-go/workflow"
	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// Order is the input payload for the OrderProcessing workflow.
type Order struct {
	ItemName  string  `json:"itemName"`
	TotalCost float64 `json:"totalCost"`
}

// OrderStatus is the slice of workflow runtime metadata exposed to callers.
type OrderStatus struct {
	InstanceID    string
	RuntimeStatus string
	Output        string
}

// ErrOrderNotFound is returned when a workflow instance is unknown to the engine.
var ErrOrderNotFound = kratoserrors.NotFound("ORDER_NOT_FOUND", "order workflow instance not found")

// OrderUsecase owns the order workflow lifecycle: it registers the workflow +
// activities, runs the worker loop, and exposes Start/Get for callers.
type OrderUsecase struct {
	client *workflow.Client
}

// NewOrderUsecase registers the order workflow + activities against the
// supplied workflow client and starts the worker loop. The returned cleanup
// func cancels the worker context on shutdown.
func NewOrderUsecase(c *workflow.Client, logger log.Logger) (*OrderUsecase, func(), error) {
	helper := log.NewHelper(logger)
	activityLog = helper
	registry := workflow.NewRegistry()
	if err := registry.AddWorkflowN(OrderWorkflowName, OrderProcessingWorkflow); err != nil {
		return nil, nil, fmt.Errorf("register workflow: %w", err)
	}
	for _, a := range []workflow.Activity{NotifyActivity, ProcessPaymentActivity} {
		if err := registry.AddActivity(a); err != nil {
			return nil, nil, fmt.Errorf("register activity: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := c.StartWorker(ctx, registry); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("start worker: %w", err)
	}
	helper.Info("dapr workflow worker started")

	cleanup := func() {
		cancel()
		helper.Info("dapr workflow worker stopped")
	}
	return &OrderUsecase{client: c}, cleanup, nil
}

func (uc *OrderUsecase) Start(ctx context.Context, o *Order) (string, error) {
	id, err := uc.client.ScheduleWorkflow(ctx, OrderWorkflowName, workflow.WithInput(o))
	if err != nil {
		return "", fmt.Errorf("schedule workflow: %w", err)
	}
	return id, nil
}

func (uc *OrderUsecase) Get(ctx context.Context, instanceID string) (*OrderStatus, error) {
	meta, err := uc.client.FetchWorkflowMetadata(ctx, instanceID, workflow.WithFetchPayloads(true))
	if err != nil {
		if errors.Is(err, api.ErrInstanceNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	return &OrderStatus{
		InstanceID:    instanceID,
		RuntimeStatus: meta.String(),
		Output:        meta.Output.GetValue(),
	}, nil
}
