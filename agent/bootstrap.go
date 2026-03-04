package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"gonesis/chat"
	"gonesis/provider"
)

var writeSoulTool = provider.Tool{
	Name:        "write_soul",
	Description: "Write your SOUL.md -- commit your identity to memory. Call this when you understand who you are.",
	Parameters: map[string]any{
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The full Markdown content for SOUL.md",
			},
		},
		"required": []any{"content"},
	},
}

// BootstrapConfig returns a chat.Config for the bootstrap interview flow.
// The executor writes SOUL.md and signals ErrDone; soulContent is populated
// via the pointer so the caller can read it after tui.Run returns.
func BootstrapConfig(ctx context.Context, p provider.Provider, baseDir string, soulContent *string) *chat.Config {
	return &chat.Config{
		Provider:     p,
		SystemPrompt: bootstrapPrompt,
		Tools:        []provider.Tool{writeSoulTool},
		Executor: func(tc provider.ToolCall) (string, error) {
			if tc.Name != "write_soul" {
				return `{"error": "unknown tool"}`, nil
			}
			content, _ := tc.Args["content"].(string)
			if content == "" {
				return `{"error": "content must not be empty"}`, nil
			}
			if err := writeSoul(baseDir, content); err != nil {
				return "", fmt.Errorf("bootstrap write: %w", err)
			}
			*soulContent = content
			return `{"status": "ok"}`, provider.ErrDone
		},
		InitialMessages: []provider.Message{
			{Role: provider.RoleUser, Content: "Hey! Let's set you up."},
		},
		WelcomeText: "Setting up a new agent...",
	}
}

func init() {
	// Ensure writeSoulTool args are valid JSON-marshalable.
	if _, err := json.Marshal(writeSoulTool.Parameters); err != nil {
		panic("invalid write_soul tool parameters: " + err.Error())
	}
}
