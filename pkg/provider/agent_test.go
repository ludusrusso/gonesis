package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ludusrusso/wildgecu/x/debug"
)

func noopLogger() *debug.Logger {
	return &debug.Logger{}
}

func TestExecuteToolsParallel(t *testing.T) {
	t.Run("RunsConcurrently", func(t *testing.T) {
		var running atomic.Int32
		var maxConcurrent atomic.Int32

		toolCalls := []ToolCall{
			{ID: "tool_a", Name: "tool_a", Args: map[string]any{"x": 1}},
			{ID: "tool_b", Name: "tool_b", Args: map[string]any{"x": 2}},
			{ID: "tool_c", Name: "tool_c", Args: map[string]any{"x": 3}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			cur := running.Add(1)
			// Track max concurrency
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			running.Add(-1)
			return "ok:" + tc.Name, nil
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if done {
			t.Fatalf("unexpected done=true")
		}

		if maxConcurrent.Load() < 2 {
			t.Errorf("expected concurrent execution, max concurrency was %d", maxConcurrent.Load())
		}

		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}

		// Results must be in the same order as toolCalls
		for i, tc := range toolCalls {
			if msgs[i].Role != RoleTool {
				t.Errorf("msgs[%d].Role = %q, want %q", i, msgs[i].Role, RoleTool)
			}
			if msgs[i].Content != "ok:"+tc.Name {
				t.Errorf("msgs[%d].Content = %q, want %q", i, msgs[i].Content, "ok:"+tc.Name)
			}
			if msgs[i].ToolCallID != tc.Name {
				t.Errorf("msgs[%d].ToolCallID = %q, want %q", i, msgs[i].ToolCallID, tc.Name)
			}
		}
	})

	t.Run("PreservesOrder", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "slow", Name: "slow", Args: map[string]any{}},
			{ID: "fast", Name: "fast", Args: map[string]any{}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			if tc.Name == "slow" {
				time.Sleep(80 * time.Millisecond)
			}
			return tc.Name + "_result", nil
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if done {
			t.Fatalf("unexpected done=true")
		}

		if msgs[0].Content != "slow_result" {
			t.Errorf("msgs[0] should be slow_result, got %q", msgs[0].Content)
		}
		if msgs[1].Content != "fast_result" {
			t.Errorf("msgs[1] should be fast_result, got %q", msgs[1].Content)
		}
	})

	t.Run("ErrDone", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "normal", Name: "normal", Args: map[string]any{}},
			{ID: "done", Name: "done", Args: map[string]any{}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			if tc.Name == "done" {
				return "finished", ErrDone
			}
			return "ok", nil
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if !done {
			t.Fatalf("expected done=true")
		}

		// All messages should still be returned
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if msgs[0].Content != "ok" {
			t.Errorf("msgs[0].Content = %q, want %q", msgs[0].Content, "ok")
		}
		if msgs[1].Content != "finished" {
			t.Errorf("msgs[1].Content = %q, want %q", msgs[1].Content, "finished")
		}
	})

	t.Run("ErrorFormatsMessage", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "failing", Name: "failing", Args: map[string]any{}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			return "", errors.New("something broke")
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if done {
			t.Fatalf("non-sentinel errors should not signal done")
		}

		if msgs[0].Content != "Error: something broke" {
			t.Errorf("expected formatted error, got %q", msgs[0].Content)
		}
	})

	t.Run("CallbackInvoked", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "a", Name: "a", Args: map[string]any{}},
			{ID: "b", Name: "b", Args: map[string]any{}},
		}

		var callbackCount atomic.Int32
		callback := ToolCallCallback(func(name, args, agent string) {
			callbackCount.Add(1)
		})

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			return "ok", nil
		}

		_, done := executeToolsParallel(context.Background(), toolCalls, executor, callback, noopLogger())
		if done {
			t.Fatalf("unexpected done=true")
		}

		if callbackCount.Load() != 2 {
			t.Errorf("expected callback called 2 times, got %d", callbackCount.Load())
		}
	})

	t.Run("EmptyToolCalls", func(t *testing.T) {
		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			t.Fatal("executor should not be called")
			return "", nil
		}

		msgs, done := executeToolsParallel(context.Background(), nil, executor, nil, noopLogger())
		if done {
			t.Fatalf("unexpected done=true")
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 messages, got %d", len(msgs))
		}
	})

	t.Run("SingleToolCall", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "only", Name: "only", Args: map[string]any{"key": "val"}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			return "single_result", nil
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if done {
			t.Fatalf("unexpected done=true")
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Content != "single_result" {
			t.Errorf("got %q, want %q", msgs[0].Content, "single_result")
		}
	})

	t.Run("MultipleErrDone", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "done1", Name: "done1", Args: map[string]any{}},
			{ID: "done2", Name: "done2", Args: map[string]any{}},
			{ID: "normal", Name: "normal", Args: map[string]any{}},
		}

		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			if tc.Name == "normal" {
				return "ok", nil
			}
			return "done", ErrDone
		}

		msgs, done := executeToolsParallel(context.Background(), toolCalls, executor, nil, noopLogger())
		if !done {
			t.Fatalf("expected done=true")
		}
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
	})
}

// mockStreamProvider implements StreamProvider for testing.
type mockStreamProvider struct {
	calls []GenerateParams
	mu    sync.Mutex
	fn    func(ctx context.Context, params *GenerateParams) (*Response, error)
}

func (m *mockStreamProvider) Generate(ctx context.Context, params *GenerateParams) (*Response, error) {
	m.mu.Lock()
	m.calls = append(m.calls, *params)
	m.mu.Unlock()
	return m.fn(ctx, params)
}

func (m *mockStreamProvider) GenerateStream(ctx context.Context, params *GenerateParams) (<-chan StreamChunk, <-chan error) {
	chunks := make(chan StreamChunk, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunks)
		resp, err := m.fn(ctx, params)
		if err != nil {
			errCh <- err
			return
		}
		m.mu.Lock()
		m.calls = append(m.calls, *params)
		m.mu.Unlock()
		chunks <- StreamChunk{
			Content:   resp.Message.Content,
			ToolCalls: resp.Message.ToolCalls,
		}
		errCh <- nil
	}()

	return chunks, errCh
}

func TestRunAgentLoopStream(t *testing.T) {
	t.Run("callback accessible from context during tool execution", func(t *testing.T) {
		// This test verifies that when RunAgentLoopStream stores the callback
		// in context, tool executors can retrieve it via GetToolCallCallback.
		// This is the code path used by spawn_agent to propagate callbacks.
		var callNum int
		sp := &mockStreamProvider{
			fn: func(_ context.Context, params *GenerateParams) (*Response, error) {
				callNum++
				if callNum == 1 {
					return &Response{
						Message: Message{
							Role:      RoleModel,
							ToolCalls: []ToolCall{{Name: "my_tool", ID: "t1", Args: map[string]any{}}},
						},
					}, nil
				}
				return &Response{
					Message: Message{Role: RoleModel, Content: "done"},
				}, nil
			},
		}

		var contextCallbackNil bool
		executor := func(ctx context.Context, tc ToolCall) (string, error) {
			cb := GetToolCallCallback(ctx)
			contextCallbackNil = (cb == nil)
			return "ok", nil
		}

		onToolCall := ToolCallCallback(func(name, args, agent string) {})
		onChunk := StreamCallback(func(chunk string) {})

		_, _, err := RunAgentLoopStream(
			context.Background(), sp, "sys",
			[]Message{{Role: RoleUser, Content: "hello"}},
			NewToolSet([]Tool{{Name: "my_tool"}}, executor),
			onChunk, onToolCall, nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contextCallbackNil {
			t.Error("GetToolCallCallback(ctx) returned nil inside tool executor — callback not propagated via context")
		}
	})

	t.Run("subagent callback propagation through streaming path", func(t *testing.T) {
		// Simulates the full subagent callback chain:
		// 1. Parent agent (streaming) makes a tool call to "spawn_agent"
		// 2. spawn_agent extracts callback from context, wraps it, calls RunAgentLoop
		// 3. Child agent makes a tool call
		// 4. Child callback fires and propagates to parent's callback with agent label
		var parentCallNum int
		parentProvider := &mockStreamProvider{
			fn: func(_ context.Context, params *GenerateParams) (*Response, error) {
				parentCallNum++
				if parentCallNum == 1 {
					return &Response{
						Message: Message{
							Role:      RoleModel,
							ToolCalls: []ToolCall{{Name: "spawn_agent", ID: "sa1", Args: map[string]any{}}},
						},
					}, nil
				}
				return &Response{
					Message: Message{Role: RoleModel, Content: "parent done"},
				}, nil
			},
		}

		var childCallNum int
		childProvider := &mockStreamProvider{
			fn: func(_ context.Context, params *GenerateParams) (*Response, error) {
				childCallNum++
				if childCallNum == 1 {
					return &Response{
						Message: Message{
							Role:      RoleModel,
							ToolCalls: []ToolCall{{Name: "child_tool", ID: "ct1", Args: map[string]any{"x": 1}}},
						},
					}, nil
				}
				return &Response{
					Message: Message{Role: RoleModel, Content: "child done"},
				}, nil
			},
		}

		type callRecord struct {
			name, args, agent string
		}
		var mu sync.Mutex
		var recorded []callRecord

		onToolCall := ToolCallCallback(func(name, args, agent string) {
			mu.Lock()
			recorded = append(recorded, callRecord{name, args, agent})
			mu.Unlock()
		})

		// Parent executor: when "spawn_agent" is called, simulate subagent behavior
		parentExecutor := func(ctx context.Context, tc ToolCall) (string, error) {
			if tc.Name == "spawn_agent" {
				// This is what subagent.go does: extract callback from context and wrap it
				parentCb := GetToolCallCallback(ctx)
				if parentCb == nil {
					return `{"error": "no callback in context"}`, nil
				}
				childOnToolCall := ToolCallCallback(func(name, args, _ string) {
					parentCb(name, args, "test-subagent")
				})

				childExecutor := func(ctx context.Context, tc ToolCall) (string, error) {
					return `{"result": "ok"}`, nil
				}

				msgs, _, err := RunAgentLoop(
					ctx, childProvider, "child system",
					[]Message{{Role: RoleUser, Content: "child task"}},
					NewToolSet([]Tool{{Name: "child_tool"}}, childExecutor),
					childOnToolCall, nil,
				)
				if err != nil {
					return "", err
				}
				for i := len(msgs) - 1; i >= 0; i-- {
					if msgs[i].Role == RoleModel && msgs[i].Content != "" {
						return msgs[i].Content, nil
					}
				}
				return "", nil
			}
			return "unknown tool", nil
		}

		onChunk := StreamCallback(func(chunk string) {})

		_, _, err := RunAgentLoopStream(
			context.Background(), parentProvider, "parent system",
			[]Message{{Role: RoleUser, Content: "run subagent"}},
			NewToolSet([]Tool{{Name: "spawn_agent"}}, parentExecutor),
			onChunk, onToolCall, nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		// We expect:
		// 1. Parent's spawn_agent call with agent="" (from executeOne in parent loop)
		// 2. Child's child_tool call with agent="test-subagent" (from executeOne in child loop, via wrapped callback)
		if len(recorded) < 2 {
			t.Fatalf("expected at least 2 callback invocations, got %d: %+v", len(recorded), recorded)
		}

		// Find the spawn_agent callback (agent should be empty — it's the parent agent's call)
		var foundSpawnAgent, foundChildTool bool
		for _, r := range recorded {
			if r.name == "spawn_agent" && r.agent == "" {
				foundSpawnAgent = true
			}
			if r.name == "child_tool" && r.agent == "test-subagent" {
				foundChildTool = true
			}
		}

		if !foundSpawnAgent {
			t.Error("expected callback for spawn_agent with empty agent label")
		}
		if !foundChildTool {
			t.Errorf("expected callback for child_tool with agent=test-subagent; recorded: %+v", recorded)
		}
	})
}
