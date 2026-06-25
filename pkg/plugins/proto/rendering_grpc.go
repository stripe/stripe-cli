// Manual gRPC service interfaces for centralized UI rendering.
// These mirror what protoc-gen-go-grpc would generate.
// TODO: Remove this file once protoc is run to regenerate main_grpc.pb.go.
package proto

import "context"

const (
	CoreCLIHelper_SendMessage_FullMethodName       = "/proto.CoreCLIHelper/SendMessage"
	CoreCLIHelper_SendCommandOutput_FullMethodName = "/proto.CoreCLIHelper/SendCommandOutput"
	CoreCLIHelper_SendProgress_FullMethodName      = "/proto.CoreCLIHelper/SendProgress"
	CoreCLIHelper_Prompt_FullMethodName            = "/proto.CoreCLIHelper/Prompt"
)

// RenderingClient extends the generated CoreCLIHelperClient with rendering methods.
type RenderingClient interface {
	SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
	SendCommandOutput(ctx context.Context, req *SendCommandOutputRequest) (*SendCommandOutputResponse, error)
	SendProgress(ctx context.Context, req *SendProgressRequest) (*SendProgressResponse, error)
	Prompt(ctx context.Context, req *PromptRequest) (*PromptResponse, error)
}
