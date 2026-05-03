package biz

import (
	"fmt"

	"github.com/dapr/durabletask-go/workflow"
)

// OrderWorkflowName is also the name used to schedule new instances.
const OrderWorkflowName = "OrderProcessingWorkflow"

// OrderProcessingWorkflow is a minimal sequential orchestration:
// notify → process payment → notify. Mirrors the pattern in
// https://docs.dapr.io/getting-started/quickstarts/workflow-quickstart/.
func OrderProcessingWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var order Order
	if err := ctx.GetInput(&order); err != nil {
		return nil, fmt.Errorf("get input: %w", err)
	}

	if err := ctx.CallActivity(NotifyActivity,
		workflow.WithActivityInput(fmt.Sprintf("Received %s ($%.2f)", order.ItemName, order.TotalCost)),
	).Await(nil); err != nil {
		return nil, err
	}

	var receipt string
	if err := ctx.CallActivity(ProcessPaymentActivity,
		workflow.WithActivityInput(order),
	).Await(&receipt); err != nil {
		return nil, err
	}

	if err := ctx.CallActivity(NotifyActivity,
		workflow.WithActivityInput(fmt.Sprintf("Fulfilled %s (%s)", order.ItemName, receipt)),
	).Await(nil); err != nil {
		return nil, err
	}

	return receipt, nil
}

// NotifyActivity stands in for sending a notification (email, slack, etc.).
func NotifyActivity(ctx workflow.ActivityContext) (any, error) {
	var msg string
	if err := ctx.GetInput(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// ProcessPaymentActivity stands in for charging a payment processor and
// returns a synthetic receipt id.
func ProcessPaymentActivity(ctx workflow.ActivityContext) (any, error) {
	var order Order
	if err := ctx.GetInput(&order); err != nil {
		return "", err
	}
	return fmt.Sprintf("receipt-%s", order.ItemName), nil
}
