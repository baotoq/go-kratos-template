package biz

import (
	"fmt"

	"github.com/dapr/durabletask-go/workflow"
	"github.com/go-kratos/kratos/v2/log"
)

// OrderWorkflowName is also the name used to schedule new instances.
const OrderWorkflowName = "OrderProcessingWorkflow"

// activityLog is set once by NewOrderUsecase so activity functions (which the
// workflow registers by reflected name) can log without taking a receiver.
var activityLog = log.NewHelper(log.DefaultLogger)

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
	activityLog.Infow("activity", "Notify", "msg", msg)
	return msg, nil
}

// ProcessPaymentActivity stands in for charging a payment processor and
// returns a synthetic receipt id.
func ProcessPaymentActivity(ctx workflow.ActivityContext) (any, error) {
	var order Order
	if err := ctx.GetInput(&order); err != nil {
		return "", err
	}
	receipt := fmt.Sprintf("receipt-%s", order.ItemName)
	activityLog.Infow("activity", "ProcessPayment", "item", order.ItemName, "total", order.TotalCost, "receipt", receipt)
	return receipt, nil
}
