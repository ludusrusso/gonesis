package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/agent/tools"
	"github.com/ludusrusso/wildgecu/pkg/provider"
)

func TestFormatTranscript(t *testing.T) {
	t.Run("KeepsUserMessagesVerbatim", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleUser, Content: "remember I prefer tabs"},
		}
		got := formatTranscript(msgs)
		if !strings.Contains(got, "**User:** remember I prefer tabs") {
			t.Fatalf("user message missing: %q", got)
		}
	})

	t.Run("TruncatesLongAssistantMessages", func(t *testing.T) {
		long := strings.Repeat("a", assistantMessageMaxChars+500)
		msgs := []provider.Message{
			{Role: provider.RoleModel, Content: long},
		}
		got := formatTranscript(msgs)
		if !strings.Contains(got, "...") {
			t.Fatalf("expected truncation marker in output: %q", got)
		}
		if len(got) > assistantMessageMaxChars+200 {
			t.Fatalf("output not truncated, got %d bytes", len(got))
		}
	})

	t.Run("CollapsesAssistantWhitespace", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleModel, Content: "line one\n\n\nline   two"},
		}
		got := formatTranscript(msgs)
		if !strings.Contains(got, "line one line two") {
			t.Fatalf("whitespace not collapsed: %q", got)
		}
	})

	t.Run("SkipsEmptyAssistantContent", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleModel, Content: "   "},
		}
		got := formatTranscript(msgs)
		if strings.Contains(got, "**Assistant:**") {
			t.Fatalf("expected empty assistant to be skipped: %q", got)
		}
	})

	t.Run("DropsDisallowedToolResults", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleModel, ToolCalls: []provider.ToolCall{{ID: "1", Name: "bash"}}},
			{Role: provider.RoleTool, ToolCallID: "1", Content: "secret output"},
		}
		got := formatTranscript(msgs)
		if strings.Contains(got, "secret output") {
			t.Fatalf("disallowed tool result should be dropped: %q", got)
		}
		if !strings.Contains(got, "called tool `bash`") {
			t.Fatalf("tool call marker missing: %q", got)
		}
	})

	t.Run("KeepsAllowlistedToolResults", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleModel, ToolCalls: []provider.ToolCall{{ID: "1", Name: "list_models"}}},
			{Role: provider.RoleTool, ToolCallID: "1", Content: "gpt-4o-mini, haiku"},
		}
		got := formatTranscript(msgs)
		if !strings.Contains(got, "gpt-4o-mini, haiku") {
			t.Fatalf("allowlisted tool result missing: %q", got)
		}
	})

	t.Run("CollapsesLongToolCallRuns", func(t *testing.T) {
		var calls []provider.ToolCall
		for i := 0; i < 5; i++ {
			calls = append(calls, provider.ToolCall{ID: "x", Name: "bash"})
		}
		msgs := []provider.Message{
			{Role: provider.RoleModel, ToolCalls: calls},
		}
		got := formatTranscript(msgs)
		if !strings.Contains(got, "×5") {
			t.Fatalf("expected collapsed marker ×5: %q", got)
		}
		if strings.Count(got, "called tool `bash`") != 1 {
			t.Fatalf("expected a single collapsed line, got: %q", got)
		}
	})

	t.Run("DoesNotCollapseShortRuns", func(t *testing.T) {
		msgs := []provider.Message{
			{Role: provider.RoleModel, ToolCalls: []provider.ToolCall{
				{ID: "a", Name: "bash"},
				{ID: "b", Name: "bash"},
			}},
		}
		got := formatTranscript(msgs)
		if strings.Contains(got, "×") {
			t.Fatalf("short run should not be collapsed: %q", got)
		}
		if strings.Count(got, "called tool `bash`") != 2 {
			t.Fatalf("expected two separate lines: %q", got)
		}
	})
}

func TestSafeTruncate(t *testing.T) {
	t.Run("ShortString", func(t *testing.T) {
		if got := safeTruncate("hi", 10); got != "hi" {
			t.Fatalf("got %q, want %q", got, "hi")
		}
	})

	t.Run("AsciiCut", func(t *testing.T) {
		if got := safeTruncate("hello world", 5); got != "hello" {
			t.Fatalf("got %q, want %q", got, "hello")
		}
	})

	t.Run("DoesNotSplitUTF8", func(t *testing.T) {
		// "caffè" — the è is two bytes (0xC3 0xA8). Cutting at 4 would split it.
		got := safeTruncate("caffè", 4)
		if got != "caff" {
			t.Fatalf("got %q (len %d), want %q", got, len(got), "caff")
		}
	})
}

func TestResolveMemoryProvider(t *testing.T) {
	t.Run("UsesDefaultWhenMemoryModelUnset", func(t *testing.T) {
		def := &fakeProvider{id: "default"}
		cfg := Config{Provider: def}
		p, label := resolveMemoryProvider(context.Background(), cfg)
		if p != def {
			t.Fatalf("expected default provider, got %v", p)
		}
		if label != "default" {
			t.Fatalf("label = %q, want %q", label, "default")
		}
	})

	t.Run("UsesResolvedWhenMemoryModelSet", func(t *testing.T) {
		def := &fakeProvider{id: "default"}
		mem := &fakeProvider{id: "memory"}
		var resolvedWith string
		cfg := Config{
			Provider:    def,
			MemoryModel: "fast",
			ResolveProvider: tools.ProviderResolver(func(_ context.Context, model string) (provider.Provider, error) {
				resolvedWith = model
				return mem, nil
			}),
		}
		p, label := resolveMemoryProvider(context.Background(), cfg)
		if p != mem {
			t.Fatalf("expected resolved provider, got %v", p)
		}
		if label != "fast" {
			t.Fatalf("label = %q, want %q", label, "fast")
		}
		if resolvedWith != "fast" {
			t.Fatalf("resolver called with %q, want %q", resolvedWith, "fast")
		}
	})

	t.Run("FallsBackWhenResolverFails", func(t *testing.T) {
		def := &fakeProvider{id: "default"}
		cfg := Config{
			Provider:    def,
			MemoryModel: "broken",
			ResolveProvider: tools.ProviderResolver(func(_ context.Context, _ string) (provider.Provider, error) {
				return nil, errors.New("nope")
			}),
		}
		p, label := resolveMemoryProvider(context.Background(), cfg)
		if p != def {
			t.Fatalf("expected fallback to default, got %v", p)
		}
		if label != "default" {
			t.Fatalf("label = %q, want %q", label, "default")
		}
	})

	t.Run("UsesDefaultWhenResolverNil", func(t *testing.T) {
		def := &fakeProvider{id: "default"}
		cfg := Config{Provider: def, MemoryModel: "fast"} // no resolver
		p, label := resolveMemoryProvider(context.Background(), cfg)
		if p != def {
			t.Fatalf("expected default provider, got %v", p)
		}
		if label != "default" {
			t.Fatalf("label = %q, want %q", label, "default")
		}
	})
}

// fakeProvider is a minimal provider.Provider stub used to verify identity.
type fakeProvider struct{ id string }

func (f *fakeProvider) Generate(_ context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	return nil, errors.New("not implemented")
}
