package tools

import (
	"context"
	"fmt"

	"github.com/ludusrusso/wildgecu/pkg/provider/tool"
	"github.com/ludusrusso/wildgecu/pkg/todo"
)

// TodoCreateName and TodoUpdateName are the registered names of the todo
// tools. They are exported so the subagent tool can exclude them from the
// default child tool set.
const (
	TodoCreateName = "todo_create"
	TodoUpdateName = "todo_update"
)

type todoCreateInput struct {
	Contents []string `json:"contents" description:"One or more todo item contents. Pass the whole initial plan atomically as a single batch."`
}

type todoSnapshotItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type todoCreateOutput struct {
	IDs  []string           `json:"ids"`
	List []todoSnapshotItem `json:"list"`
}

type todoUpdateInput struct {
	ID      string `json:"id" description:"ID of the todo item to update (echoed from a prior todo_create response)."`
	Content string `json:"content,omitempty" description:"Optional new content. Omit to leave unchanged."`
	Status  string `json:"status,omitempty" description:"Optional new status: pending, in_progress, completed, cancelled. Omit to leave unchanged."`
}

type todoUpdateOutput struct {
	List []todoSnapshotItem `json:"list"`
}

// TodoTools returns the todo_create and todo_update tools. The handlers
// resolve the active session's *todo.List via todo.ListFromContext.
func TodoTools() []tool.Tool {
	return []tool.Tool{todoCreateTool, todoUpdateTool}
}

var todoCreateTool = tool.NewTool(TodoCreateName,
	"Create one or more todo items in the session-scoped todo list. Pass the entire initial plan atomically as a single batch — do NOT call this once per item. Returns the assigned IDs in input order plus the full current list.",
	func(ctx context.Context, in todoCreateInput) (todoCreateOutput, error) {
		l := todo.ListFromContext(ctx)
		if l == nil {
			return todoCreateOutput{}, fmt.Errorf("todo list not available in this context")
		}
		ids, err := l.Create(in.Contents...)
		if err != nil {
			return todoCreateOutput{}, err
		}
		return todoCreateOutput{IDs: ids, List: snapshot(l)}, nil
	},
)

var todoUpdateTool = tool.NewTool(TodoUpdateName,
	"Update the content and/or status of a todo item. Status values: pending, in_progress, completed, cancelled. Returns the full current list.",
	func(ctx context.Context, in todoUpdateInput) (todoUpdateOutput, error) {
		l := todo.ListFromContext(ctx)
		if l == nil {
			return todoUpdateOutput{}, fmt.Errorf("todo list not available in this context")
		}
		if in.ID == "" {
			return todoUpdateOutput{}, fmt.Errorf("id is required")
		}
		var contentPtr *string
		if in.Content != "" {
			c := in.Content
			contentPtr = &c
		}
		if err := l.Update(in.ID, contentPtr, todo.Status(in.Status)); err != nil {
			return todoUpdateOutput{}, err
		}
		return todoUpdateOutput{List: snapshot(l)}, nil
	},
)

func snapshot(l *todo.List) []todoSnapshotItem {
	items := l.Snapshot()
	out := make([]todoSnapshotItem, len(items))
	for i, it := range items {
		out[i] = todoSnapshotItem{ID: it.ID, Content: it.Content, Status: string(it.Status)}
	}
	return out
}
