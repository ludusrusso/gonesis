package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ludusrusso/wildgecu/pkg/home"
	"github.com/ludusrusso/wildgecu/pkg/provider"
	"github.com/ludusrusso/wildgecu/pkg/provider/tool"
)

// WriteMemoryInput is the input for the write_memory tool.
type WriteMemoryInput struct {
	Content string `json:"content" description:"The full Markdown content for MEMORY.md"`
}

// WriteMemoryOutput is the output for the write_memory tool.
type WriteMemoryOutput struct {
	Status string `json:"status"`
}

// Transcript compression tuning.
const (
	assistantMessageMaxChars = 1500
	toolResultKeepMaxChars   = 300
	consecutiveToolCollapse  = 3
)

// toolResultAllowlist is the small set of tools whose results we keep in the
// transcript — their output is often the fact worth remembering. Everything
// else is dropped to save tokens.
var toolResultAllowlist = map[string]struct{}{
	"read_memory": {},
	"list_models": {},
}

// RunMemoryAgent reviews the conversation and updates MEMORY.md.
// modelLabel is used only for observability logs.
func RunMemoryAgent(ctx context.Context, p provider.Provider, modelLabel string, h *home.Home, messages []provider.Message, currentMemory string) error {
	transcript := formatTranscript(messages)

	start := time.Now()
	slog.Info("memory agent: start",
		"model", modelLabel,
		"messages", len(messages),
		"memory_bytes", len(currentMemory),
		"transcript_bytes", len(transcript),
		"raw_content_bytes", rawContentBytes(messages),
	)
	defer func() {
		slog.Info("memory agent: done", "elapsed", time.Since(start))
	}()

	writeMemoryTool := tool.NewTool("write_memory",
		"Write the updated MEMORY.md content.",
		func(ctx context.Context, in WriteMemoryInput) (WriteMemoryOutput, error) {
			if in.Content == "" {
				return WriteMemoryOutput{}, fmt.Errorf("content must not be empty")
			}
			if err := h.Memory().Write(in.Content); err != nil {
				return WriteMemoryOutput{}, fmt.Errorf("writing MEMORY.md: %w", err)
			}
			slog.Info("memory agent: write_memory invoked", "bytes", len(in.Content))
			return WriteMemoryOutput{Status: "ok"}, provider.ErrDone
		},
	)

	registry := tool.NewRegistry(writeMemoryTool)

	var userMsg strings.Builder
	userMsg.WriteString("## Conversation Transcript\n\n")
	userMsg.WriteString(transcript)
	userMsg.WriteString("\n\n## Current MEMORY.md\n\n")
	if currentMemory == "" {
		userMsg.WriteString("(empty — no existing memory)")
	} else {
		userMsg.WriteString(currentMemory)
	}
	userMsg.WriteString("\n\nReview the conversation above and call `write_memory` with the updated memory content.")

	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: userMsg.String()},
	}

	_, _, err := provider.RunAgentLoop(ctx, p, memoryAgentPrompt, msgs, registry.Tools(), registry.Executor(), nil, nil)
	if err != nil && !errors.Is(err, provider.ErrDone) {
		return fmt.Errorf("memory agent: %w", err)
	}
	return nil
}

// formatTranscript renders the conversation into a compact form for the memory
// agent. It preserves user messages verbatim (highest-signal), truncates long
// assistant messages, drops tool results (except a small allowlist), and
// collapses runs of identical consecutive tool calls.
func formatTranscript(messages []provider.Message) string {
	var b strings.Builder

	// Collect linear entries first so we can collapse tool-call runs in a
	// second pass. Each entry is either a rendered line or a tool-call marker
	// we may merge with neighbors.
	type entry struct {
		kind string // "text" or "toolcall"
		name string // tool name (for toolcall)
		text string // pre-rendered text (for text)
	}
	var entries []entry

	// Tool-result messages only carry ToolCallID, not the tool name. Track
	// id → name as we walk model messages so we can filter tool results by
	// name.
	toolNames := map[string]string{}

	for _, m := range messages {
		switch m.Role {
		case provider.RoleUser:
			entries = append(entries, entry{kind: "text", text: fmt.Sprintf("**User:** %s", m.Content)})
		case provider.RoleModel:
			if c := compactAssistant(m.Content); c != "" {
				entries = append(entries, entry{kind: "text", text: fmt.Sprintf("**Assistant:** %s", c)})
			}
			for _, tc := range m.ToolCalls {
				toolNames[tc.ID] = tc.Name
				entries = append(entries, entry{kind: "toolcall", name: tc.Name})
			}
		case provider.RoleTool:
			name := toolNames[m.ToolCallID]
			if _, keep := toolResultAllowlist[name]; !keep {
				continue
			}
			result := m.Content
			if len(result) > toolResultKeepMaxChars {
				result = safeTruncate(result, toolResultKeepMaxChars) + "..."
			}
			entries = append(entries, entry{kind: "text", text: fmt.Sprintf("**Tool result (%s):** %s", name, result)})
		}
	}

	// Collapse consecutive identical tool calls when the run exceeds the
	// threshold.
	i := 0
	for i < len(entries) {
		e := entries[i]
		if e.kind != "toolcall" {
			b.WriteString(e.text)
			b.WriteString("\n\n")
			i++
			continue
		}
		j := i + 1
		for j < len(entries) && entries[j].kind == "toolcall" && entries[j].name == e.name {
			j++
		}
		n := j - i
		if n > consecutiveToolCollapse {
			fmt.Fprintf(&b, "**Assistant** called tool `%s` ×%d\n\n", e.name, n)
		} else {
			for k := 0; k < n; k++ {
				fmt.Fprintf(&b, "**Assistant** called tool `%s`\n\n", e.name)
			}
		}
		i = j
	}

	return b.String()
}

// compactAssistant collapses internal whitespace and truncates long assistant
// messages. Empty input returns empty output so the caller can skip the entry.
func compactAssistant(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > assistantMessageMaxChars {
		s = safeTruncate(s, assistantMessageMaxChars) + "..."
	}
	return s
}

// rawContentBytes is a rough size-of-transcript metric used only for logs.
func rawContentBytes(messages []provider.Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content)
	}
	return total
}

// safeTruncate cuts s to at most n bytes without splitting a UTF-8 code point.
func safeTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Walk back from n until we land on a UTF-8 start byte (top bits != 10).
	for n > 0 && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}
