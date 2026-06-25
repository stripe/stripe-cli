// Manually written proto message types for centralized UI rendering.
// These mirror what protoc-gen-go would generate from main.proto.
// TODO: Remove this file once protoc is run to regenerate main.pb.go.
package proto

type MessageLevel int32

const (
	MessageLevel_INFO    MessageLevel = 0
	MessageLevel_SUCCESS MessageLevel = 1
	MessageLevel_WARNING MessageLevel = 2
	MessageLevel_ERROR   MessageLevel = 3
)

type ProgressType int32

const (
	ProgressType_PROGRESS_INFO  ProgressType = 0
	ProgressType_STEP           ProgressType = 1
	ProgressType_SPINNER_START  ProgressType = 2
	ProgressType_SPINNER_UPDATE ProgressType = 3
	ProgressType_SPINNER_STOP   ProgressType = 4
)

type PromptType int32

const (
	PromptType_TEXT    PromptType = 0
	PromptType_CONFIRM PromptType = 1
	PromptType_SELECT  PromptType = 2
)

type SendMessageRequest struct {
	Message string       `json:"message"`
	Level   MessageLevel `json:"level"`
}

type SendMessageResponse struct{}

// OutputBlock is a typed block within a CommandOutputRequest.
// Blocks are rendered in order. Type determines how payload is interpreted.
type OutputBlock struct {
	Type    string `json:"type"`    // "data", "warning", "nextstep", "error"
	Payload string `json:"payload"` // JSON-encoded payload (shape varies per type)
}

type SendCommandOutputRequest struct {
	Command string         `json:"command"`
	Blocks  []*OutputBlock `json:"blocks"`
	// NOTE: `ok` field intentionally omitted. Errors go to stderr via SendMessage
	// (level=ERROR) + non-zero exit code. Adding `ok` later is not a breaking change.
}

type SendCommandOutputResponse struct{}

type SendProgressRequest struct {
	Id      string       `json:"id"`
	Message string       `json:"message"`
	Type    ProgressType `json:"type"`
	Success bool         `json:"success"` // relevant on SPINNER_STOP only
}

type SendProgressResponse struct{}

type PromptRequest struct {
	Message      string     `json:"message"`
	Type         PromptType `json:"type"`
	Options      []string   `json:"options"`
	DefaultValue string     `json:"default_value"`
}

type PromptResponse struct {
	Value string `json:"value"`
}
