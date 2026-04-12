package tools

import (
	"context"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/provider"
)

func TestInformTools(t *testing.T) {
	tools := InformTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 inform tool, got %d", len(tools))
	}
	if got := tools[0].Definition().Name; got != "inform_user" {
		t.Errorf("expected tool name inform_user, got %q", got)
	}
}

func TestInformUser(t *testing.T) {
	tl := InformTools()[0]

	t.Run("calls callback with message", func(t *testing.T) {
		var received string
		ctx := provider.WithInformFunc(context.Background(), func(msg string) {
			received = msg
		})
		_, err := tl.Execute(ctx, map[string]any{"message": "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if received != "hello" {
			t.Errorf("expected callback to receive %q, got %q", "hello", received)
		}
	})

	t.Run("no panic without callback", func(t *testing.T) {
		_, err := tl.Execute(context.Background(), map[string]any{"message": "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestInformContext(t *testing.T) {
	t.Run("round-trip", func(t *testing.T) {
		called := false
		fn := provider.InformFunc(func(string) { called = true })
		ctx := provider.WithInformFunc(context.Background(), fn)
		got := provider.GetInformFunc(ctx)
		if got == nil {
			t.Fatal("expected non-nil InformFunc from context")
		}
		got("test")
		if !called {
			t.Error("expected InformFunc to be called")
		}
	})

	t.Run("nil when not set", func(t *testing.T) {
		got := provider.GetInformFunc(context.Background())
		if got != nil {
			t.Error("expected nil InformFunc from bare context")
		}
	})
}
