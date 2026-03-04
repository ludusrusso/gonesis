package chat

import (
	"context"

	"gonesis/provider"
)

// Config holds everything needed to run a conversational loop.
type Config struct {
	Provider     provider.Provider
	SystemPrompt string
	Tools        []provider.Tool
	Executor     provider.ToolExecutor

	// InitialMessages, if set, are sent to the model before any user input.
	// This lets the model speak first (e.g. bootstrap interview).
	InitialMessages []provider.Message

	// WelcomeText is displayed at the top of the chat UI.
	WelcomeText string

	// OnDone is called when the executor signals provider.ErrDone.
	// It receives the final message history.
	OnDone func(messages []provider.Message)
}

// RunTurn appends a user message to the conversation and runs one agent loop.
// It returns the updated messages and the model's response.
func RunTurn(ctx context.Context, cfg *Config, messages []provider.Message, userInput string) ([]provider.Message, *provider.Response, error) {
	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: userInput,
	})
	return provider.RunAgentLoop(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.Tools, cfg.Executor)
}

// RunInitialTurn runs the agent loop on pre-seeded messages without adding a user message.
func RunInitialTurn(ctx context.Context, cfg *Config, messages []provider.Message) ([]provider.Message, *provider.Response, error) {
	return provider.RunAgentLoop(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.Tools, cfg.Executor)
}

// RunTurnStream is like RunTurn but streams text chunks via onChunk.
func RunTurnStream(ctx context.Context, cfg *Config, messages []provider.Message, userInput string, onChunk provider.StreamCallback) ([]provider.Message, *provider.Response, error) {
	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: userInput,
	})
	return provider.RunAgentLoopStream(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.Tools, cfg.Executor, onChunk)
}

// RunInitialTurnStream is like RunInitialTurn but streams text chunks via onChunk.
func RunInitialTurnStream(ctx context.Context, cfg *Config, messages []provider.Message, onChunk provider.StreamCallback) ([]provider.Message, *provider.Response, error) {
	return provider.RunAgentLoopStream(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.Tools, cfg.Executor, onChunk)
}
