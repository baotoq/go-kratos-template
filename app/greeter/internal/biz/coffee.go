package biz

import (
	"context"
	"errors"
	"fmt"

	"github.com/dapr/durabletask-go/api"
	"github.com/dapr/durabletask-go/workflow"
	kratoserrors "github.com/go-kratos/kratos/v2/errors"
)

// CoffeeOrder is what you tell the workflow you want.
type CoffeeOrder struct {
	Beans string `json:"beans"` // e.g. "arabica"
	Size  string `json:"size"`  // "small", "medium", "large"
}

// CoffeeStatus is what you get back when you peek at a workflow instance.
type CoffeeStatus struct {
	InstanceID string
	Status     string // RUNNING, COMPLETED, FAILED, ...
	Cup        string // the finished coffee, once Status == "COMPLETED"
}

// ErrCoffeeNotFound is returned when no workflow instance exists for the id.
var ErrCoffeeNotFound = kratoserrors.NotFound("COFFEE_NOT_FOUND", "coffee instance not found")

// CoffeeUsecase wires the workflow client into a tidy Brew/Check API.
type CoffeeUsecase struct {
	client *workflow.Client
}

// NewCoffeeUsecase registers the workflow + activities and starts the worker.
// The cleanup func cancels the worker context on shutdown.
func NewCoffeeUsecase(c *workflow.Client) (*CoffeeUsecase, func(), error) {
	registry := workflow.NewRegistry()
	if err := registry.AddWorkflowN(CoffeeWorkflowName, MakeCoffeeWorkflow); err != nil {
		return nil, nil, fmt.Errorf("register workflow: %w", err)
	}
	for _, a := range []workflow.Activity{GrindBeansActivity, BrewActivity} {
		if err := registry.AddActivity(a); err != nil {
			return nil, nil, fmt.Errorf("register activity: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := c.StartWorker(ctx, registry); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("start worker: %w", err)
	}
	cleanup := func() { cancel() }
	return &CoffeeUsecase{client: c}, cleanup, nil
}

// Brew schedules a new MakeCoffeeWorkflow instance and returns its id.
func (uc *CoffeeUsecase) Brew(ctx context.Context, o *CoffeeOrder) (string, error) {
	id, err := uc.client.ScheduleWorkflow(ctx, CoffeeWorkflowName, workflow.WithInput(o))
	if err != nil {
		return "", fmt.Errorf("schedule workflow: %w", err)
	}
	return id, nil
}

// Check fetches the current state of a workflow instance.
func (uc *CoffeeUsecase) Check(ctx context.Context, instanceID string) (*CoffeeStatus, error) {
	meta, err := uc.client.FetchWorkflowMetadata(ctx, instanceID, workflow.WithFetchPayloads(true))
	if err != nil {
		if errors.Is(err, api.ErrInstanceNotFound) {
			return nil, ErrCoffeeNotFound
		}
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	return &CoffeeStatus{
		InstanceID: instanceID,
		Status:     meta.String(),
		Cup:        meta.Output.GetValue(),
	}, nil
}
