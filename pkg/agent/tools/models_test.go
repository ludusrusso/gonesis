package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestModelTools(t *testing.T) {
	t.Run("nil info returns no tools", func(t *testing.T) {
		tools := ModelTools(nil)
		if len(tools) != 0 {
			t.Fatalf("expected 0 tools, got %d", len(tools))
		}
	})

	t.Run("returns one tool named list_models", func(t *testing.T) {
		info := &ModelInfo{DefaultModel: "gemini/gemini-2.5-flash"}
		tools := ModelTools(info)
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		if got := tools[0].Definition().Name; got != "list_models" {
			t.Errorf("expected tool name list_models, got %q", got)
		}
	})
}

func TestListModels(t *testing.T) {
	t.Run("returns configured model info", func(t *testing.T) {
		info := &ModelInfo{
			DefaultModel: "gemini/gemini-2.5-flash",
			Providers:    []string{"openai", "gemini"},
			Models: map[string]string{
				"fast":  "gemini/gemini-2.0-flash",
				"smart": "gemini/gemini-2.5-pro",
			},
		}
		tl := ModelTools(info)[0]
		result, err := tl.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var out ModelInfo
		if err := json.Unmarshal([]byte(result), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.DefaultModel != "gemini/gemini-2.5-flash" {
			t.Errorf("expected default_model %q, got %q", "gemini/gemini-2.5-flash", out.DefaultModel)
		}
		if len(out.Providers) != 2 {
			t.Fatalf("expected 2 providers, got %d", len(out.Providers))
		}
		if out.Providers[0] != "gemini" || out.Providers[1] != "openai" {
			t.Errorf("expected sorted providers [gemini openai], got %v", out.Providers)
		}
		if out.Models["fast"] != "gemini/gemini-2.0-flash" {
			t.Errorf("expected alias fast=%q, got %q", "gemini/gemini-2.0-flash", out.Models["fast"])
		}
		if out.Models["smart"] != "gemini/gemini-2.5-pro" {
			t.Errorf("expected alias smart=%q, got %q", "gemini/gemini-2.5-pro", out.Models["smart"])
		}
	})

	t.Run("handles nil models map", func(t *testing.T) {
		info := &ModelInfo{
			DefaultModel: "ollama/llama3",
			Providers:    []string{"ollama"},
		}
		tl := ModelTools(info)[0]
		result, err := tl.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var out ModelInfo
		if err := json.Unmarshal([]byte(result), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.DefaultModel != "ollama/llama3" {
			t.Errorf("expected default_model %q, got %q", "ollama/llama3", out.DefaultModel)
		}
		if len(out.Models) != 0 {
			t.Errorf("expected empty models, got %v", out.Models)
		}
	})

	t.Run("does not mutate original providers order", func(t *testing.T) {
		info := &ModelInfo{
			DefaultModel: "a/b",
			Providers:    []string{"z", "a", "m"},
		}
		tl := ModelTools(info)[0]
		_, err := tl.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Providers[0] != "z" || info.Providers[1] != "a" || info.Providers[2] != "m" {
			t.Errorf("original providers were mutated: %v", info.Providers)
		}
	})
}
