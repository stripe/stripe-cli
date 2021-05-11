package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/logtailing"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
)

var run func() error

type mockTailer struct {
}

func (mt *mockTailer) Run(ctx context.Context) error {
	return run()
}

func TestLogsTailStreamsLogs(t *testing.T) {
	ctx := withAuth(context.Background())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	logtailingOutCh := make(chan logtailing.IElement)

	run = func() error {
		logtailingOutCh <- logtailing.StateElement{
			State: logtailing.Loading,
		}
		logtailingOutCh <- logtailing.StateElement{
			State: logtailing.Reconnecting,
		}
		logtailingOutCh <- logtailing.StateElement{
			State: logtailing.Ready,
		}
		logtailingOutCh <- logtailing.StateElement{
			State: logtailing.Done,
		}
		return nil
	}

	createTailer = func(cfg *logtailing.Config) ITailer {
		return &mockTailer{}
	}

	logsTailClient, err := client.LogsTail(ctx)
	assert.Nil(t, err)

	logsTailClient.Send(&rpc.LogsTailRequest{})

	resp, err := logsTailClient.Recv()

	assert.Nil(t, err.Error())
	// assert.Nil(t, resp.GetLog())
	assert.Nil(t, resp.GetState())
}
