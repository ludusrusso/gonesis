package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/todo"
)

func execCreate(ctx context.Context, t *testing.T, args map[string]any) todoCreateOutput {
	t.Helper()
	raw, err := todoCreateTool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("todo_create: %v", err)
	}
	var out todoCreateOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal: %v (raw=%s)", err, raw)
	}
	return out
}

func execUpdate(ctx context.Context, t *testing.T, args map[string]any) todoUpdateOutput {
	t.Helper()
	raw, err := todoUpdateTool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("todo_update: %v", err)
	}
	var out todoUpdateOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal: %v (raw=%s)", err, raw)
	}
	return out
}

func TestTodoTools(t *testing.T) {
	t.Run("BatchCreateThenUpdateReflectsInSnapshot", func(t *testing.T) {
		l := todo.New()
		ctx := todo.WithList(context.Background(), l)

		out := execCreate(ctx, t, map[string]any{"contents": []any{"a", "b", "c"}})
		if len(out.IDs) != 3 || out.IDs[0] != "1" || out.IDs[1] != "2" || out.IDs[2] != "3" {
			t.Fatalf("expected sequential ids, got %v", out.IDs)
		}
		if len(out.List) != 3 || out.List[0].Content != "a" || out.List[2].Content != "c" {
			t.Fatalf("snapshot order wrong: %#v", out.List)
		}

		upd := execUpdate(ctx, t, map[string]any{"id": "2", "status": "completed"})
		if upd.List[1].Status != "completed" {
			t.Fatalf("expected completed, got %q", upd.List[1].Status)
		}
		if upd.List[0].Status != "pending" || upd.List[2].Status != "pending" {
			t.Fatal("unrelated items should stay pending")
		}
	})

	t.Run("SingleElementBatchBehavesLikeMultiElement", func(t *testing.T) {
		l := todo.New()
		ctx := todo.WithList(context.Background(), l)
		out := execCreate(ctx, t, map[string]any{"contents": []any{"only"}})
		if len(out.IDs) != 1 || out.IDs[0] != "1" {
			t.Fatalf("expected single id [1], got %v", out.IDs)
		}
		if len(out.List) != 1 || out.List[0].Content != "only" {
			t.Fatalf("unexpected list: %#v", out.List)
		}
	})

	t.Run("UnknownIDReturnsError", func(t *testing.T) {
		l := todo.New()
		ctx := todo.WithList(context.Background(), l)
		raw, err := todoUpdateTool.Execute(ctx, map[string]any{"id": "999", "status": "completed"})
		if err == nil {
			t.Fatalf("expected error, got %s", raw)
		}
		if !strings.Contains(err.Error(), "unknown id") {
			t.Fatalf("expected unknown id error, got %v", err)
		}
	})

	t.Run("InvalidStatusReturnsError", func(t *testing.T) {
		l := todo.New()
		ctx := todo.WithList(context.Background(), l)
		execCreate(ctx, t, map[string]any{"contents": []any{"x"}})
		_, err := todoUpdateTool.Execute(ctx, map[string]any{"id": "1", "status": "bogus"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("MissingListInContextReturnsError", func(t *testing.T) {
		_, err := todoCreateTool.Execute(context.Background(), map[string]any{"contents": []any{"x"}})
		if err == nil {
			t.Fatal("expected error when list missing")
		}
	})
}
