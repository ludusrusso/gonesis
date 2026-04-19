package command

import (
	"context"
	"fmt"

	"github.com/ludusrusso/wildgecu/pkg/todo"
)

// TodosLookupFunc returns the *todo.List for the given session ID, or an
// error if the session is unknown.
type TodosLookupFunc func(ctx context.Context, sessionID string) (*todo.List, error)

// TodosCommand prints the current session-scoped todo list.
type TodosCommand struct {
	lookup TodosLookupFunc
}

// NewTodosCommand creates a /todos command backed by lookup.
func NewTodosCommand(lookup TodosLookupFunc) *TodosCommand {
	return &TodosCommand{lookup: lookup}
}

func (c *TodosCommand) Name() string        { return "todos" }
func (c *TodosCommand) Description() string { return "Show the current session todo list" }

func (c *TodosCommand) Execute(ctx context.Context, _ string) (string, error) {
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return "", fmt.Errorf("no active session")
	}
	list, err := c.lookup(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if list == nil || list.IsEmpty() {
		return "No active todos.", nil
	}
	return list.RenderMarkdown(), nil
}
