package provider

// ToolSet is what the agent loop needs from a collection of tools: the
// definitions to advertise to the model, and an executor to dispatch calls.
// *tool.Registry satisfies this interface; NewToolSet wraps a bare
// (tools, executor) pair for ad-hoc callers.
type ToolSet interface {
	Tools() []Tool
	Executor() ToolExecutor
}

// NewToolSet adapts a tool slice and executor into a ToolSet. Either may be
// nil; the agent loop treats nil/empty tools as "no tools" and never calls a
// nil executor (it's only invoked when the model emits a tool call).
func NewToolSet(tools []Tool, execute ToolExecutor) ToolSet {
	return &toolSet{tools: tools, execute: execute}
}

type toolSet struct {
	tools   []Tool
	execute ToolExecutor
}

func (t *toolSet) Tools() []Tool         { return t.tools }
func (t *toolSet) Executor() ToolExecutor { return t.execute }

// unpackToolSet is nil-safe: a nil ToolSet behaves as no tools / no executor.
func unpackToolSet(ts ToolSet) ([]Tool, ToolExecutor) {
	if ts == nil {
		return nil, nil
	}
	return ts.Tools(), ts.Executor()
}
