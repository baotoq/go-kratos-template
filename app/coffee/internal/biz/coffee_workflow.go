package biz

import (
	"fmt"
	"log"

	"github.com/dapr/durabletask-go/workflow"
)

// CoffeeWorkflowName is the registered orchestration name.
const CoffeeWorkflowName = "MakeCoffeeWorkflow"

// MakeCoffeeWorkflow is a two-step workflow:
//  1. Grind the beans.
//  2. Brew using the grounds + the requested size.
func MakeCoffeeWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var order CoffeeOrder
	if err := ctx.GetInput(&order); err != nil {
		return nil, err
	}

	// Step 1: grind.
	var grounds string
	if err := ctx.CallActivity(GrindBeansActivity,
		workflow.WithActivityInput(order.Beans),
	).Await(&grounds); err != nil {
		return nil, err
	}

	// Step 2: brew, using the grounds from step 1 + the size from the order.
	var cup string
	if err := ctx.CallActivity(BrewActivity,
		workflow.WithActivityInput(BrewInput{Grounds: grounds, Size: order.Size}),
	).Await(&cup); err != nil {
		return nil, err
	}

	return cup, nil
}

// BrewInput is what step 2 receives: the grounds from step 1 + the requested size.
type BrewInput struct {
	Grounds string `json:"grounds"`
	Size    string `json:"size"`
}

// GrindBeansActivity grinds whatever beans you give it.
func GrindBeansActivity(actx workflow.ActivityContext) (any, error) {
	var beans string
	if err := actx.GetInput(&beans); err != nil {
		return nil, err
	}
	log.Printf("coffee: grinding %s beans", beans)
	return fmt.Sprintf("ground %s", beans), nil
}

// BrewActivity turns grounds into a finished cup of coffee.
func BrewActivity(actx workflow.ActivityContext) (any, error) {
	var in BrewInput
	if err := actx.GetInput(&in); err != nil {
		return nil, err
	}
	log.Printf("coffee: brewing a %s cup from %s", in.Size, in.Grounds)
	return fmt.Sprintf("a %s cup of %s coffee", in.Size, in.Grounds), nil
}
