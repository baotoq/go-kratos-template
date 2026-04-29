package data

import (
	"net"
	"os"

	dapr "github.com/dapr/go-sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewDaprClient() (dapr.Client, func(), error) {
	port := os.Getenv("DAPR_GRPC_PORT")
	if port == "" {
		port = "50001"
	}
	addr := net.JoinHostPort("127.0.0.1", port)

	// grpc.NewClient does not block — connection is established lazily on first RPC.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	client := dapr.NewClientWithConnection(conn)
	cleanup := func() {
		client.Close()
	}
	return client, cleanup, nil
}
