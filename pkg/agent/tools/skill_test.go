package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const testSkillContent = `---
name: test-skill
description: "A test skill"
tags:
  - testing
---
## Test Skill Content
This is the body.`

const testSkill2Content = `---
name: another-skill
description: "Another skill"
---
Second skill body.`

func TestSkillTools(t *testing.T) {
	t.Run("empty dir returns nil", func(t *testing.T) {
		tools := SkillTools("")
		if tools != nil {
			t.Fatalf("expected nil for empty skillsDir, got %d tools", len(tools))
		}
	})

	t.Run("returns load_skill tool", func(t *testing.T) {
		tools := SkillTools("/tmp")
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		if tools[0].Definition().Name != "load_skill" {
			t.Fatalf("tool name = %q", tools[0].Definition().Name)
		}
	})
}

func TestLoadSkill(t *testing.T) {
	t.Run("list empty dir", func(t *testing.T) {
		dir := t.TempDir()
		tl := newLoadSkillTool(dir)

		var out loadSkillOutput
		result, err := tl.Execute(context.Background(), map[string]any{"action": "list"})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if len(out.Skills) != 0 {
			t.Fatalf("expected 0 skills, got %d", len(out.Skills))
		}
	})

	t.Run("list multiple skills", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)
		writeSkill(t, dir, "another-skill", testSkill2Content)

		tl := newLoadSkillTool(dir)
		var out loadSkillOutput
		result, _ := tl.Execute(context.Background(), map[string]any{"action": "list"})
		json.Unmarshal([]byte(result), &out)

		if len(out.Skills) != 2 {
			t.Fatalf("expected 2 skills, got %d", len(out.Skills))
		}
		names := map[string]bool{}
		for _, s := range out.Skills {
			names[s.Name] = true
		}
		if !names["test-skill"] || !names["another-skill"] {
			t.Fatalf("unexpected skill names: %v", names)
		}
	})

	t.Run("list includes tags", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)

		tl := newLoadSkillTool(dir)
		var out loadSkillOutput
		result, _ := tl.Execute(context.Background(), map[string]any{"action": "list"})
		json.Unmarshal([]byte(result), &out)

		if len(out.Skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(out.Skills))
		}
		if len(out.Skills[0].Tags) != 1 || out.Skills[0].Tags[0] != "testing" {
			t.Fatalf("tags = %v, want [testing]", out.Skills[0].Tags)
		}
	})

	t.Run("load by name", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)

		tl := newLoadSkillTool(dir)
		var out loadSkillOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"action": "load",
			"name":   "test-skill",
		})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if out.Name != "test-skill" {
			t.Fatalf("name = %q", out.Name)
		}
		if out.Content == "" {
			t.Fatal("expected non-empty content")
		}
	})

	t.Run("load missing name", func(t *testing.T) {
		dir := t.TempDir()
		tl := newLoadSkillTool(dir)

		_, err := tl.Execute(context.Background(), map[string]any{
			"action": "load",
			"name":   "",
		})
		if err == nil {
			t.Fatal("expected error for missing name")
		}
	})

	t.Run("load nonexistent skill", func(t *testing.T) {
		dir := t.TempDir()
		tl := newLoadSkillTool(dir)

		_, err := tl.Execute(context.Background(), map[string]any{
			"action": "load",
			"name":   "no-such-skill",
		})
		if err == nil {
			t.Fatal("expected error for nonexistent skill")
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		dir := t.TempDir()
		tl := newLoadSkillTool(dir)

		_, err := tl.Execute(context.Background(), map[string]any{
			"action": "delete",
		})
		if err == nil {
			t.Fatal("expected error for unknown action")
		}
	})
}
