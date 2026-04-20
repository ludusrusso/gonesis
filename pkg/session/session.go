package session

import (
	"context"

	"github.com/ludusrusso/wildgecu/x/debug"
	"github.com/ludusrusso/wildgecu/pkg/provider"
)

// Config holds everything needed to run a conversational loop.
type Config struct {
	Provider        provider.Provider
	Executor        provider.ToolExecutor
	OnDone          func(messages []provider.Message)
	OnToolCall      provider.ToolCallCallback
	Debug           *debug.Logger
	SystemPrompt    string
	WelcomeText     string
	Tools           []provider.Tool
	InitialMessages []provider.Message

	// RequestReminder, when set, returns a string appended to the tail user
	// message content for the duration of the outgoing provider call. The
	// appended text is stripped before the returned messages slice is handed
	// back to the caller, so it never lands in the session's canonical log.
	// Used to inject session-scoped <system-reminder> blocks without
	// invalidating prompt cache.
	RequestReminder func() string
}

// RunTurn appends a user message to the conversation and runs one agent loop.
// If userInput is empty the user message is omitted.
func RunTurn(ctx context.Context, cfg *Config, messages []provider.Message, userInput string) ([]provider.Message, *provider.Response, error) {
	if userInput != "" {
		cfg.Debug.UserMessage(userInput)
		messages = append(messages, provider.Message{
			Role:    provider.RoleUser,
			Content: userInput,
		})
	}
	transient, tailIdx := applyReminder(messages, cfg)
	updated, resp, err := provider.RunAgentLoop(ctx, cfg.Provider, cfg.SystemPrompt, transient, cfg.toolSet(), cfg.OnToolCall, cfg.Debug)
	stripReminder(updated, messages, tailIdx)
	return updated, resp, err
}

// RunInitialTurn runs the agent loop on pre-seeded messages without adding a user message.
func RunInitialTurn(ctx context.Context, cfg *Config, messages []provider.Message) ([]provider.Message, *provider.Response, error) {
	return provider.RunAgentLoop(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.toolSet(), cfg.OnToolCall, cfg.Debug)
}

// RunTurnStream is like RunTurn but streams text chunks via onChunk.
// If userInput is empty the user message is omitted (useful for skill
// commands where the system prompt alone drives the response).
func RunTurnStream(ctx context.Context, cfg *Config, messages []provider.Message, userInput string, onChunk provider.StreamCallback) ([]provider.Message, *provider.Response, error) {
	if userInput != "" {
		cfg.Debug.UserMessage(userInput)
		messages = append(messages, provider.Message{
			Role:    provider.RoleUser,
			Content: userInput,
		})
	}
	transient, tailIdx := applyReminder(messages, cfg)
	updated, resp, err := provider.RunAgentLoopStream(ctx, cfg.Provider, cfg.SystemPrompt, transient, cfg.toolSet(), onChunk, cfg.OnToolCall, cfg.Debug)
	stripReminder(updated, messages, tailIdx)
	return updated, resp, err
}

// RunInitialTurnStream is like RunInitialTurn but streams text chunks via onChunk.
func RunInitialTurnStream(ctx context.Context, cfg *Config, messages []provider.Message, onChunk provider.StreamCallback) ([]provider.Message, *provider.Response, error) {
	return provider.RunAgentLoopStream(ctx, cfg.Provider, cfg.SystemPrompt, messages, cfg.toolSet(), onChunk, cfg.OnToolCall, cfg.Debug)
}

// toolSet wraps the Config's Tools+Executor as a provider.ToolSet. Built per
// call so that a caller who swaps cfg.Executor (e.g. sessions.go wrapping it
// for todo snapshots) still has its updated executor picked up.
func (c *Config) toolSet() provider.ToolSet {
	return provider.NewToolSet(c.Tools, c.Executor)
}

// applyReminder returns a transient messages slice that carries the reminder
// appended to the tail user message, plus the index of that user message. If
// no reminder is active, it returns the input slice unchanged and -1.
func applyReminder(messages []provider.Message, cfg *Config) ([]provider.Message, int) {
	if cfg.RequestReminder == nil {
		return messages, -1
	}
	reminder := cfg.RequestReminder()
	if reminder == "" {
		return messages, -1
	}
	tail := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == provider.RoleUser {
			tail = i
			break
		}
	}
	if tail < 0 {
		return messages, -1
	}
	transient := make([]provider.Message, len(messages))
	copy(transient, messages)
	transient[tail].Content = transient[tail].Content + "\n\n" + reminder
	return transient, tail
}

// stripReminder restores the tail user message in updated to its pre-reminder
// form. This guarantees that the caller's persisted log never contains the
// reminder text.
func stripReminder(updated, original []provider.Message, tailIdx int) {
	if tailIdx < 0 || tailIdx >= len(updated) || tailIdx >= len(original) {
		return
	}
	updated[tailIdx] = original[tailIdx]
}
