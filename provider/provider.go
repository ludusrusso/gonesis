package provider

import "context"

type Role string

const (
	RoleUser  Role = "user"
	RoleModel Role = "model"
	RoleTool  Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string // set when Role == RoleTool
}

// ToolCall represents a function call requested by the model.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// Tool defines a function the model can call.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema object
}

// GenerateParams holds everything needed for a single generation request.
type GenerateParams struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Response is what comes back from a generation call.
type Response struct {
	Message Message
	Usage   Usage
}

// Provider is the interface every LLM backend must implement.
type Provider interface {
	Generate(ctx context.Context, params *GenerateParams) (*Response, error)
}

// StreamChunk is a partial text chunk from a streaming response.
type StreamChunk struct {
	Content   string
	ToolCalls []ToolCall // populated in the last chunk if the model made tool calls
	Usage     Usage      // populated on last chunk
}

// StreamProvider extends Provider with streaming support.
type StreamProvider interface {
	Provider
	GenerateStream(ctx context.Context, params *GenerateParams) (<-chan StreamChunk, <-chan error)
}

// StreamCallback is called for each text chunk during a streaming response.
type StreamCallback func(chunk string)
