package data

import (
	"github.com/dapr/durabletask-go/workflow"
	dapr "github.com/dapr/go-sdk/client"
)

// NewWorkflowClient constructs a durabletask workflow client over the existing
// Dapr sidecar gRPC connection. The grpc.ClientConn lifecycle is owned by the
// dapr.Client passed in (closed by main on shutdown).
func NewWorkflowClient(daprClient dapr.Client) *workflow.Client {
	return workflow.NewClient(daprClient.GrpcClientConn())
}
