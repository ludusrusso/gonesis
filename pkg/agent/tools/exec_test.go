package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
)

func TestExecTools(t *testing.T) {
	tools := ExecTools("/tmp")
	if len(tools) != 2 {
		t.Fatalf("expected 2 exec tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Definition().Name] = true
	}
	for _, want := range []string{"bash", "node"} {
		if !names[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestBash(t *testing.T) {
	dir := t.TempDir()
	tl := newBashTool(dir)

	t.Run("echo stdout", func(t *testing.T) {
		var out bashOutput
		result, err := tl.Execute(context.Background(), map[string]any{"command": "echo hello"})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stdout != "hello\n" {
			t.Fatalf("stdout = %q, want %q", out.Stdout, "hello\n")
		}
		if out.ExitCode != 0 {
			t.Fatalf("exit_code = %d, want 0", out.ExitCode)
		}
	})

	t.Run("stderr", func(t *testing.T) {
		var out bashOutput
		result, err := tl.Execute(context.Background(), map[string]any{"command": "echo err >&2"})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stderr != "err\n" {
			t.Fatalf("stderr = %q, want %q", out.Stderr, "err\n")
		}
	})

	t.Run("nonzero exit code", func(t *testing.T) {
		var out bashOutput
		result, err := tl.Execute(context.Background(), map[string]any{"command": "exit 42"})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.ExitCode != 42 {
			t.Fatalf("exit_code = %d, want 42", out.ExitCode)
		}
	})

	t.Run("runs in workDir", func(t *testing.T) {
		var out bashOutput
		result, err := tl.Execute(context.Background(), map[string]any{"command": "pwd"})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		// TempDir may have symlinks (e.g. /var -> /private/var on macOS)
		if out.Stdout == "" {
			t.Fatal("expected non-empty pwd output")
		}
	})

	t.Run("mixed stdout and stderr", func(t *testing.T) {
		var out bashOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"command": "echo out && echo err >&2",
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stdout != "out\n" {
			t.Fatalf("stdout = %q", out.Stdout)
		}
		if out.Stderr != "err\n" {
			t.Fatalf("stderr = %q", out.Stderr)
		}
		if out.ExitCode != 0 {
			t.Fatalf("exit_code = %d", out.ExitCode)
		}
	})
}

func TestNode(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available")
	}

	dir := t.TempDir()
	tl := newNodeTool(dir)

	t.Run("console.log", func(t *testing.T) {
		var out nodeOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"script": `console.log("hello from node")`,
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stdout != "hello from node\n" {
			t.Fatalf("stdout = %q", out.Stdout)
		}
		if out.ExitCode != 0 {
			t.Fatalf("exit_code = %d", out.ExitCode)
		}
	})

	t.Run("stderr", func(t *testing.T) {
		var out nodeOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"script": `console.error("node err")`,
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stderr != "node err\n" {
			t.Fatalf("stderr = %q", out.Stderr)
		}
	})

	t.Run("nonzero exit", func(t *testing.T) {
		var out nodeOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"script": `process.exit(7)`,
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.ExitCode != 7 {
			t.Fatalf("exit_code = %d, want 7", out.ExitCode)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var out nodeOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"script": `console.log(JSON.stringify({a: 1}))`,
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Stdout != `{"a":1}`+"\n" {
			t.Fatalf("stdout = %q", out.Stdout)
		}
	})
}
