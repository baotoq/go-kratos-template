package biz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/stretchr/testify/assert"
)

// fakeActivityContext is a minimal in-memory stub of workflow.ActivityContext
// (which aliases task.ActivityContext) that hands back a JSON-serialised input.
type fakeActivityContext struct {
	input any
}

func (f *fakeActivityContext) GetInput(out any) error {
	data, err := json.Marshal(f.input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (*fakeActivityContext) GetTaskID() int32                     { return 0 }
func (*fakeActivityContext) GetTaskExecutionID() string           { return "" }
func (*fakeActivityContext) Context() context.Context             { return context.Background() }
func (*fakeActivityContext) GetTraceContext() *protos.TraceContext { return nil }

func TestGrindBeansActivity_grinds_the_beans(t *testing.T) {
	// Arrange
	ctx := workflow.ActivityContext(&fakeActivityContext{input: "arabica"})

	// Act
	got, err := GrindBeansActivity(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "ground arabica", got)
}

func TestBrewActivity_returns_a_finished_cup(t *testing.T) {
	// Arrange
	ctx := workflow.ActivityContext(&fakeActivityContext{
		input: BrewInput{Grounds: "ground arabica", Size: "large"},
	})

	// Act
	got, err := BrewActivity(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "a large cup of ground arabica coffee", got)
}
