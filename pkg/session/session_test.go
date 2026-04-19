package session

import (
	"context"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/provider"
)

// stubProvider captures the messages sent on Generate and returns a no-op
// model response (no tool calls).
type stubProvider struct {
	lastMessages []provider.Message
	lastSystem   string
}

func (s *stubProvider) Generate(_ context.Context, params *provider.GenerateParams) (*provider.Response, error) {
	s.lastMessages = append([]provider.Message{}, params.Messages...)
	s.lastSystem = params.SystemPrompt
	return &provider.Response{
		Message: provider.Message{Role: provider.RoleModel, Content: "ok"},
	}, nil
}

func TestRunTurnReminder(t *testing.T) {
	t.Run("ReminderAppendedToOutgoingTailUserMessageButNotPersisted", func(t *testing.T) {
		p := &stubProvider{}
		cfg := &Config{
			Provider:        p,
			SystemPrompt:    "sys",
			
			RequestReminder: func() string { return "<system-reminder>\nTODO\n</system-reminder>" },
		}
		updated, _, err := RunTurn(context.Background(), cfg, nil, "hello")
		if err != nil {
			t.Fatalf("RunTurn: %v", err)
		}

		if len(p.lastMessages) == 0 {
			t.Fatal("provider received no messages")
		}
		tail := p.lastMessages[len(p.lastMessages)-1]
		if tail.Role != provider.RoleUser {
			t.Fatalf("expected tail user message, got %s", tail.Role)
		}
		if !strings.Contains(tail.Content, "<system-reminder>") {
			t.Fatalf("outgoing tail user content missing reminder: %q", tail.Content)
		}
		if !strings.HasPrefix(tail.Content, "hello") {
			t.Fatalf("outgoing tail user content should start with original input: %q", tail.Content)
		}

		var userMsg *provider.Message
		for i := range updated {
			if updated[i].Role == provider.RoleUser {
				userMsg = &updated[i]
			}
		}
		if userMsg == nil {
			t.Fatal("expected persisted user message")
		}
		if strings.Contains(userMsg.Content, "<system-reminder>") {
			t.Fatalf("persisted user message should not contain reminder: %q", userMsg.Content)
		}
		if userMsg.Content != "hello" {
			t.Fatalf("persisted user content should match input, got %q", userMsg.Content)
		}
	})

	t.Run("NoReminderLeavesMessagesUnchanged", func(t *testing.T) {
		p := &stubProvider{}
		cfg := &Config{Provider: p, SystemPrompt: "sys"}
		_, _, err := RunTurn(context.Background(), cfg, nil, "hi")
		if err != nil {
			t.Fatalf("RunTurn: %v", err)
		}
		tail := p.lastMessages[len(p.lastMessages)-1]
		if tail.Content != "hi" {
			t.Fatalf("expected %q, got %q", "hi", tail.Content)
		}
	})

	t.Run("SystemPromptIsByteIdenticalAcrossTurnsWhenReminderChanges", func(t *testing.T) {
		p := &stubProvider{}
		reminder := "one"
		cfg := &Config{
			Provider:        p,
			SystemPrompt:    "stable prompt",
			
			RequestReminder: func() string { return reminder },
		}
		messages, _, err := RunTurn(context.Background(), cfg, nil, "a")
		if err != nil {
			t.Fatalf("RunTurn 1: %v", err)
		}
		sys1 := p.lastSystem

		reminder = "two-and-different"
		_, _, err = RunTurn(context.Background(), cfg, messages, "b")
		if err != nil {
			t.Fatalf("RunTurn 2: %v", err)
		}
		sys2 := p.lastSystem
		if sys1 != sys2 {
			t.Fatalf("system prompt drifted: %q vs %q", sys1, sys2)
		}
	})
}
