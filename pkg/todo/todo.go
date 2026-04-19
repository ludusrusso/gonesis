// Package todo implements a session-scoped, in-memory todo list used by the
// agent to externalize its multi-step plan. The package has no dependencies
// on session, agent, tool, TUI, or daemon layers.
package todo

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Status is the lifecycle state of a todo item.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

// Valid reports whether s is a known status value.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// Item is a single todo entry.
type Item struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// List is the session-scoped todo list. The zero value is ready to use.
type List struct {
	mu    sync.Mutex
	items []Item
	next  int
	now   func() time.Time
}

// New returns an empty list.
func New() *List {
	return &List{}
}

func (l *List) clock() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

// Create appends one or more items to the list and returns the newly assigned
// IDs in input order. Empty or all-blank input is rejected.
func (l *List) Create(contents ...string) ([]string, error) {
	if len(contents) == 0 {
		return nil, fmt.Errorf("todo: create requires at least one item")
	}
	trimmed := make([]string, len(contents))
	for i, c := range contents {
		t := strings.TrimSpace(c)
		if t == "" {
			return nil, fmt.Errorf("todo: item %d has empty content", i)
		}
		trimmed[i] = t
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	ids := make([]string, len(trimmed))
	for i, c := range trimmed {
		l.next++
		id := strconv.Itoa(l.next)
		l.items = append(l.items, Item{
			ID:        id,
			Content:   c,
			Status:    StatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		})
		ids[i] = id
	}
	return ids, nil
}

// Update mutates the content and/or status of the item with the given ID.
// Pass a nil content or empty status to leave that field unchanged.
func (l *List) Update(id string, content *string, status Status) error {
	if content != nil {
		if strings.TrimSpace(*content) == "" {
			return fmt.Errorf("todo: content must not be blank")
		}
	}
	if status != "" && !status.Valid() {
		return fmt.Errorf("todo: invalid status %q", string(status))
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.items {
		if l.items[i].ID != id {
			continue
		}
		if content != nil {
			l.items[i].Content = strings.TrimSpace(*content)
		}
		if status != "" {
			l.items[i].Status = status
		}
		l.items[i].UpdatedAt = l.clock()
		return nil
	}
	return fmt.Errorf("todo: unknown id %q", id)
}

// Snapshot returns a defensive copy of the current items.
func (l *List) Snapshot() []Item {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Item, len(l.items))
	copy(out, l.items)
	return out
}

// IsEmpty reports whether the list has no items.
func (l *List) IsEmpty() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.items) == 0
}

// RenderMarkdown returns a Markdown checklist of the current items. Returns
// the empty string when the list is empty.
func (l *List) RenderMarkdown() string {
	items := l.Snapshot()
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "- %s %s", checkbox(it.Status), it.Content)
	}
	return b.String()
}

// RenderSystemReminder returns a <system-reminder> block listing the current
// items for ephemeral injection into the outgoing user message. Returns the
// empty string when the list is empty.
func (l *List) RenderSystemReminder() string {
	items := l.Snapshot()
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<system-reminder>\n")
	b.WriteString("Current todo list (session-scoped). Keep statuses up to date via todo_update.\n")
	for _, it := range items {
		fmt.Fprintf(&b, "- [%s] #%s %s\n", it.Status, it.ID, it.Content)
	}
	b.WriteString("</system-reminder>")
	return b.String()
}

func checkbox(s Status) string {
	switch s {
	case StatusCompleted:
		return "[x]"
	case StatusInProgress:
		return "[~]"
	case StatusCancelled:
		return "[-]"
	default:
		return "[ ]"
	}
}
