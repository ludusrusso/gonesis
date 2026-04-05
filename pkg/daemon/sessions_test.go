package daemon

import (
	"context"
	"testing"

	"wildgecu/pkg/agent"
	"wildgecu/pkg/home"
	"wildgecu/pkg/provider"
	"wildgecu/pkg/session"
)

// fakeProvider returns a canned response without calling any real LLM.
type fakeProvider struct{}

func (fakeProvider) Generate(_ context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	return &provider.Response{
		Message: provider.Message{Role: "assistant", Content: "ok"},
	}, nil
}

func newTestSessionManager(t *testing.T) *SessionManager {
	t.Helper()
	h, err := home.New(t.TempDir())
	if err != nil {
		t.Fatalf("home.New: %v", err)
	}
	return &SessionManager{
		agentCfg: agent.Config{
			Provider: fakeProvider{},
			Home:     h,
		},
		chatCfg: &session.Config{
			Provider:    fakeProvider{},
			WelcomeText: "hello",
		},
		sessions: make(map[string]*ManagedSession),
	}
}

func TestReset(t *testing.T) {
	t.Run("returns new session with different ID", func(t *testing.T) {
		sm := newTestSessionManager(t)
		old := sm.Create()
		oldID := old.ID

		// Add a message so the old session is non-empty.
		old.Messages = append(old.Messages, provider.Message{Role: "user", Content: "hi"})

		newSess, err := sm.Reset(context.Background(), oldID)
		if err != nil {
			t.Fatalf("Reset() error: %v", err)
		}
		if newSess.ID == oldID {
			t.Error("expected new session to have a different ID")
		}
	})

	t.Run("old session is removed", func(t *testing.T) {
		sm := newTestSessionManager(t)
		old := sm.Create()
		oldID := old.ID

		_, err := sm.Reset(context.Background(), oldID)
		if err != nil {
			t.Fatalf("Reset() error: %v", err)
		}
		if sm.Get(oldID) != nil {
			t.Error("expected old session to be removed")
		}
	})

	t.Run("new session is retrievable", func(t *testing.T) {
		sm := newTestSessionManager(t)
		old := sm.Create()

		newSess, err := sm.Reset(context.Background(), old.ID)
		if err != nil {
			t.Fatalf("Reset() error: %v", err)
		}
		if sm.Get(newSess.ID) == nil {
			t.Error("expected new session to be retrievable")
		}
	})

	t.Run("new session has fresh messages", func(t *testing.T) {
		sm := newTestSessionManager(t)
		sm.chatCfg.InitialMessages = []provider.Message{
			{Role: "system", Content: "You are helpful."},
		}
		old := sm.Create()
		// Simulate conversation history.
		old.Messages = append(old.Messages,
			provider.Message{Role: "user", Content: "hello"},
			provider.Message{Role: "assistant", Content: "hi there"},
		)

		newSess, err := sm.Reset(context.Background(), old.ID)
		if err != nil {
			t.Fatalf("Reset() error: %v", err)
		}
		// New session should only have the initial messages, not the old conversation.
		if len(newSess.Messages) != 1 {
			t.Errorf("expected 1 initial message, got %d", len(newSess.Messages))
		}
		if newSess.Messages[0].Content != "You are helpful." {
			t.Errorf("expected initial system message, got %q", newSess.Messages[0].Content)
		}
	})

	t.Run("error on unknown session", func(t *testing.T) {
		sm := newTestSessionManager(t)

		_, err := sm.Reset(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown session")
		}
	})
}
