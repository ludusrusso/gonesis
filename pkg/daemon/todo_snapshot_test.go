package daemon

import (
	"context"
	"sync"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/agent/tools"
	"github.com/ludusrusso/wildgecu/pkg/provider"
	"github.com/ludusrusso/wildgecu/pkg/session"
	"github.com/ludusrusso/wildgecu/pkg/todo"
)

// todoToolProvider makes one todo_create tool call then returns text.
type todoToolProvider struct {
	callNum int
}

func (p *todoToolProvider) Generate(_ context.Context, params *provider.GenerateParams) (*provider.Response, error) {
	p.callNum++
	if p.callNum == 1 {
		return &provider.Response{
			Message: provider.Message{
				Role: provider.RoleModel,
				ToolCalls: []provider.ToolCall{
					{Name: tools.TodoCreateName, ID: "t1", Args: map[string]any{"contents": []any{"step one", "step two"}}},
				},
			},
		}, nil
	}
	return &provider.Response{Message: provider.Message{Role: provider.RoleModel, Content: "done"}}, nil
}

func TestRunTurnStreamEmitsTodoSnapshot(t *testing.T) {
	t.Run("SnapshotEmittedAfterTodoCreateExecutes", func(t *testing.T) {
		executor := func(ctx context.Context, tc provider.ToolCall) (string, error) {
			l := todo.ListFromContext(ctx)
			if l == nil {
				t.Fatal("todo list missing from context")
			}
			if _, err := l.Create("step one", "step two"); err != nil {
				t.Fatalf("Create: %v", err)
			}
			return "ok", nil
		}

		p := &todoToolProvider{}
		sm := &SessionManager{
			chatCfg: &session.Config{
				Provider:     p,
				SystemPrompt: "test",
				Tools:        []provider.Tool{{Name: tools.TodoCreateName}},
				Executor:     executor,
			},
			sessions: make(map[string]*ManagedSession),
		}
		sess := sm.Create()

		var mu sync.Mutex
		var snapshots [][]todo.Item
		onTodoSnapshot := func(items []todo.Item) {
			mu.Lock()
			defer mu.Unlock()
			snapshots = append(snapshots, items)
		}

		if _, err := sm.RunTurnStream(context.Background(), sess.ID, "hello", nil, nil, nil, onTodoSnapshot); err != nil {
			t.Fatalf("RunTurnStream: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(snapshots) != 1 {
			t.Fatalf("expected exactly 1 snapshot emission, got %d", len(snapshots))
		}
		if got := len(snapshots[0]); got != 2 {
			t.Errorf("snapshot item count = %d, want 2", got)
		}
	})

	t.Run("NoSnapshotEmitterWhenCallbackNil", func(t *testing.T) {
		executor := func(ctx context.Context, tc provider.ToolCall) (string, error) {
			l := todo.ListFromContext(ctx)
			if l != nil {
				_, _ = l.Create("only")
			}
			return "ok", nil
		}

		p := &todoToolProvider{}
		sm := &SessionManager{
			chatCfg: &session.Config{
				Provider: p,
				Tools:    []provider.Tool{{Name: tools.TodoCreateName}},
				Executor: executor,
			},
			sessions: make(map[string]*ManagedSession),
		}
		sess := sm.Create()

		if _, err := sm.RunTurnStream(context.Background(), sess.ID, "hello", nil, nil, nil, nil); err != nil {
			t.Fatalf("RunTurnStream: %v", err)
		}
	})
}
