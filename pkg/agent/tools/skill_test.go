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

	t.Run("returns list_skills and read_skill tools", func(t *testing.T) {
		tools := SkillTools("/tmp")
		if len(tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(tools))
		}
		if tools[0].Definition().Name != "list_skills" {
			t.Fatalf("first tool name = %q, want list_skills", tools[0].Definition().Name)
		}
		if tools[1].Definition().Name != "read_skill" {
			t.Fatalf("second tool name = %q, want read_skill", tools[1].Definition().Name)
		}
	})
}

func TestListSkills(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		dir := t.TempDir()
		tl := newListSkillsTool(dir)

		var out listSkillsOutput
		result, err := tl.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal([]byte(result), &out)

		if len(out.Skills) != 0 {
			t.Fatalf("expected 0 skills, got %d", len(out.Skills))
		}
	})

	t.Run("multiple skills", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)
		writeSkill(t, dir, "another-skill", testSkill2Content)

		tl := newListSkillsTool(dir)
		var out listSkillsOutput
		result, _ := tl.Execute(context.Background(), map[string]any{})
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

	t.Run("includes tags", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)

		tl := newListSkillsTool(dir)
		var out listSkillsOutput
		result, _ := tl.Execute(context.Background(), map[string]any{})
		json.Unmarshal([]byte(result), &out)

		if len(out.Skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(out.Skills))
		}
		if len(out.Skills[0].Tags) != 1 || out.Skills[0].Tags[0] != "testing" {
			t.Fatalf("tags = %v, want [testing]", out.Skills[0].Tags)
		}
	})
}

func TestReadSkill(t *testing.T) {
	t.Run("load by name", func(t *testing.T) {
		dir := t.TempDir()
		writeSkill(t, dir, "test-skill", testSkillContent)

		tl := newReadSkillTool(dir)
		var out readSkillOutput
		result, err := tl.Execute(context.Background(), map[string]any{
			"name": "test-skill",
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

	t.Run("missing name", func(t *testing.T) {
		dir := t.TempDir()
		tl := newReadSkillTool(dir)

		_, err := tl.Execute(context.Background(), map[string]any{
			"name": "",
		})
		if err == nil {
			t.Fatal("expected error for missing name")
		}
	})

	t.Run("nonexistent skill", func(t *testing.T) {
		dir := t.TempDir()
		tl := newReadSkillTool(dir)

		_, err := tl.Execute(context.Background(), map[string]any{
			"name": "no-such-skill",
		})
		if err == nil {
			t.Fatal("expected error for nonexistent skill")
		}
	})
}
