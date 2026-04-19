package command

import (
	"context"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/todo"
)

func TestTodosCommand(t *testing.T) {
	t.Run("EmptyListReturnsNoActiveTodos", func(t *testing.T) {
		l := todo.New()
		cmd := NewTodosCommand(func(_ context.Context, _ string) (*todo.List, error) { return l, nil })
		ctx := WithSessionID(context.Background(), "s1")
		got, err := cmd.Execute(ctx, "")
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got != "No active todos." {
			t.Fatalf("expected 'No active todos.', got %q", got)
		}
	})

	t.Run("NonEmptyListReturnsMarkdown", func(t *testing.T) {
		l := todo.New()
		l.Create("first task", "second task")
		cmd := NewTodosCommand(func(_ context.Context, _ string) (*todo.List, error) { return l, nil })
		ctx := WithSessionID(context.Background(), "s1")
		got, err := cmd.Execute(ctx, "")
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if !strings.Contains(got, "first task") || !strings.Contains(got, "second task") {
			t.Fatalf("expected items in output, got %q", got)
		}
	})

	t.Run("MissingSessionReturnsError", func(t *testing.T) {
		cmd := NewTodosCommand(func(_ context.Context, _ string) (*todo.List, error) { return nil, nil })
		if _, err := cmd.Execute(context.Background(), ""); err == nil {
			t.Fatal("expected error when no session id in context")
		}
	})
}
