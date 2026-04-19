package todo

import (
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	t.Run("CreateSingleItemReturnsID", func(t *testing.T) {
		l := New()
		ids, err := l.Create("write docs")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if len(ids) != 1 || ids[0] != "1" {
			t.Fatalf("expected [\"1\"], got %v", ids)
		}
		items := l.Snapshot()
		if len(items) != 1 || items[0].Content != "write docs" || items[0].Status != StatusPending {
			t.Fatalf("unexpected items: %#v", items)
		}
	})

	t.Run("CreateBatchPreservesInputOrderAndAssignsUniqueSequentialIDs", func(t *testing.T) {
		l := New()
		ids, err := l.Create("a", "b", "c")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if len(ids) != 3 || ids[0] != "1" || ids[1] != "2" || ids[2] != "3" {
			t.Fatalf("expected [\"1\",\"2\",\"3\"], got %v", ids)
		}
		items := l.Snapshot()
		if items[0].Content != "a" || items[1].Content != "b" || items[2].Content != "c" {
			t.Fatalf("unexpected content order: %#v", items)
		}
	})

	t.Run("CreateRejectsEmptyInput", func(t *testing.T) {
		l := New()
		if _, err := l.Create(); err == nil {
			t.Fatal("expected error on empty input")
		}
	})

	t.Run("CreateRejectsBlankContent", func(t *testing.T) {
		l := New()
		if _, err := l.Create("ok", "   "); err == nil {
			t.Fatal("expected error on blank content")
		}
		if len(l.Snapshot()) != 0 {
			t.Fatal("expected no partial insert on error")
		}
	})

	t.Run("UpdateChangesContentWithoutTouchingStatus", func(t *testing.T) {
		l := New()
		ids, _ := l.Create("old")
		newContent := "new"
		if err := l.Update(ids[0], &newContent, ""); err != nil {
			t.Fatalf("Update: %v", err)
		}
		items := l.Snapshot()
		if items[0].Content != "new" || items[0].Status != StatusPending {
			t.Fatalf("unexpected item: %#v", items[0])
		}
	})

	t.Run("UpdateChangesStatusWithoutTouchingContent", func(t *testing.T) {
		l := New()
		ids, _ := l.Create("task")
		if err := l.Update(ids[0], nil, StatusInProgress); err != nil {
			t.Fatalf("Update: %v", err)
		}
		items := l.Snapshot()
		if items[0].Content != "task" || items[0].Status != StatusInProgress {
			t.Fatalf("unexpected item: %#v", items[0])
		}
	})

	t.Run("UpdateUnknownIDReturnsError", func(t *testing.T) {
		l := New()
		if err := l.Update("999", nil, StatusCompleted); err == nil {
			t.Fatal("expected error on unknown id")
		}
	})

	t.Run("UpdateInvalidStatusReturnsError", func(t *testing.T) {
		l := New()
		ids, _ := l.Create("x")
		if err := l.Update(ids[0], nil, Status("bogus")); err == nil {
			t.Fatal("expected error on invalid status")
		}
	})

	t.Run("SnapshotIsDefensiveCopy", func(t *testing.T) {
		l := New()
		l.Create("a")
		snap := l.Snapshot()
		snap[0].Content = "mutated"
		if l.Snapshot()[0].Content != "a" {
			t.Fatal("mutating snapshot leaked into list")
		}
	})

	t.Run("RenderMarkdownPerStatusUsesDocumentedGlyphs", func(t *testing.T) {
		l := New()
		ids, _ := l.Create("p", "w", "d", "c")
		_ = l.Update(ids[1], nil, StatusInProgress)
		_ = l.Update(ids[2], nil, StatusCompleted)
		_ = l.Update(ids[3], nil, StatusCancelled)
		md := l.RenderMarkdown()
		for _, want := range []string{"- [ ] p", "- [~] w", "- [x] d", "- [-] c"} {
			if !strings.Contains(md, want) {
				t.Fatalf("markdown missing %q:\n%s", want, md)
			}
		}
	})

	t.Run("RenderSystemReminderEmptyListReturnsEmptyString", func(t *testing.T) {
		l := New()
		if got := l.RenderSystemReminder(); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("RenderSystemReminderNonEmptyHasTaggedBlock", func(t *testing.T) {
		l := New()
		l.Create("alpha")
		got := l.RenderSystemReminder()
		if !strings.HasPrefix(got, "<system-reminder>") || !strings.HasSuffix(got, "</system-reminder>") {
			t.Fatalf("expected tagged block, got %q", got)
		}
		if !strings.Contains(got, "alpha") || !strings.Contains(got, "#1") {
			t.Fatalf("expected item listed, got %q", got)
		}
	})

	t.Run("IsEmptyTransitions", func(t *testing.T) {
		l := New()
		if !l.IsEmpty() {
			t.Fatal("new list should be empty")
		}
		l.Create("x")
		if l.IsEmpty() {
			t.Fatal("list with item should not be empty")
		}
	})
}
