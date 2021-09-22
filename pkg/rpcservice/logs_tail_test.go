package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/logtailing"
	"github.com/stripe/stripe-cli/pkg/websocket"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var run func(ctx context.Context) error

type mockTailer struct {
	OutCh chan websocket.IElement
}

func (mt *mockTailer) Run(ctx context.Context) error {
	return run(ctx)
}

func TestLogsTailStreamsState(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createTailer = func(cfg *logtailing.Config) ITailer {
		run = func(ctx context.Context) error {
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Loading,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Reconnecting,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Ready,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Done,
			}
			return nil
		}
		return &mockTailer{
			OutCh: cfg.OutCh,
		}
	}

	logsTailClient, err := client.LogsTail(ctx, &rpc.LogsTailRequest{})
	assert.Nil(t, err)

	expectedStates := []rpc.LogsTailResponse_State{
		rpc.LogsTailResponse_STATE_LOADING,
		rpc.LogsTailResponse_STATE_RECONNECTING,
		rpc.LogsTailResponse_STATE_READY,
		rpc.LogsTailResponse_STATE_DONE,
	}

	for _, s := range expectedStates {
		resp, err := logsTailClient.Recv()
		assert.Nil(t, err)
		assert.Equal(t, s, resp.GetState())
	}

	cancel()

	resp, err := logsTailClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestLogsTailStreamsLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createTailer = func(cfg *logtailing.Config) ITailer {
		run = func(ctx context.Context) error {
			cfg.OutCh <- websocket.DataElement{
				Data: logtailing.EventPayload{
					RequestID: "req_1",
				},
			}
			cfg.OutCh <- websocket.DataElement{
				Data: logtailing.EventPayload{
					RequestID: "req_2",
					Error: logtailing.RedactedError{
						Message: "my error",
					},
				},
			}
			return nil
		}
		return &mockTailer{
			OutCh: cfg.OutCh,
		}
	}

	logsTailClient, err := client.LogsTail(ctx, &rpc.LogsTailRequest{})
	assert.Nil(t, err)

	expectedLogs := []rpc.LogsTailResponse_Log{
		{
			RequestId: "req_1",
			Error:     nil,
		},
		{
			RequestId: "req_2",
			Error:     &rpc.LogsTailResponse_Log_Error{Message: "my error"},
		},
	}

	for i := range expectedLogs {
		resp, err := logsTailClient.Recv()
		assert.Nil(t, err)
		assert.Equal(t, &expectedLogs[i], resp.GetLog())
	}

	cancel()

	resp, err := logsTailClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestLogsTailReturnsError(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createTailer = func(cfg *logtailing.Config) ITailer {
		run = func(ctx context.Context) error {
			myErr := errors.New("my error")
			cfg.OutCh <- websocket.ErrorElement{
				Error: myErr,
			}
			return myErr
		}
		return &mockTailer{
			OutCh: cfg.OutCh,
		}
	}

	logsTailClient, err := client.LogsTail(ctx, &rpc.LogsTailRequest{})
	assert.Nil(t, err)

	resp, err := logsTailClient.Recv()
	assert.Equal(t, status.Error(codes.Unknown, "my error").Error(), err.Error())
	assert.Nil(t, resp)
}
