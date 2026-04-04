package skill

import (
	"strings"
	"testing"

	"wildgecu/x/home"
)

func TestParseValid(t *testing.T) {
	data := []byte("---\nname: go-errors\ndescription: \"Go error handling\"\ntags:\n  - go\n  - errors\n---\n## Best Practices\nWrap errors.")
	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Name != "go-errors" {
		t.Errorf("expected name go-errors, got %q", s.Name)
	}
	if s.Description != "Go error handling" {
		t.Errorf("expected description 'Go error handling', got %q", s.Description)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "go" || s.Tags[1] != "errors" {
		t.Errorf("expected tags [go errors], got %v", s.Tags)
	}
	if s.Content != "## Best Practices\nWrap errors." {
		t.Errorf("unexpected content: %q", s.Content)
	}
}

func TestParseMissingFrontmatter(t *testing.T) {
	_, err := Parse([]byte("no frontmatter here"))
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseMissingClosingDelimiter(t *testing.T) {
	_, err := Parse([]byte("---\nname: test\ndescription: test\n"))
	if err == nil {
		t.Fatal("expected error for missing closing delimiter")
	}
}

func TestParseMissingName(t *testing.T) {
	_, err := Parse([]byte("---\ndescription: test\n---\ncontent"))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseMissingDescription(t *testing.T) {
	_, err := Parse([]byte("---\nname: test\n---\ncontent"))
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	original := &Skill{
		Name:        "go-errors",
		Description: "Go error handling best practices",
		Tags:        []string{"go", "errors"},
		Content:     "## Best Practices\nWrap errors.",
	}

	data, err := Serialize(original)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse round-trip failed: %v", err)
	}

	if parsed.Name != original.Name {
		t.Errorf("name mismatch: %q vs %q", parsed.Name, original.Name)
	}
	if parsed.Description != original.Description {
		t.Errorf("description mismatch: %q vs %q", parsed.Description, original.Description)
	}
	if len(parsed.Tags) != len(original.Tags) {
		t.Errorf("tags length mismatch: %d vs %d", len(parsed.Tags), len(original.Tags))
	}
	if parsed.Content != original.Content {
		t.Errorf("content mismatch: %q vs %q", parsed.Content, original.Content)
	}
}

func TestSerializeNoTags(t *testing.T) {
	s := &Skill{
		Name:        "simple",
		Description: "A simple skill",
		Content:     "Some content",
	}

	data, err := Serialize(s)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if strings.Contains(string(data), "tags:") {
		t.Error("expected no tags field in output")
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse round-trip failed: %v", err)
	}
	if parsed.Name != s.Name {
		t.Errorf("name mismatch: %q vs %q", parsed.Name, s.Name)
	}
}

func TestSkillPath(t *testing.T) {
	got := SkillPath("go-errors")
	if got != "go-errors/SKILL.md" {
		t.Errorf("expected go-errors/SKILL.md, got %q", got)
	}
}

func TestLoadAll(t *testing.T) {
	h := home.NewMem()

	h.Upsert("good/SKILL.md", []byte("---\nname: good\ndescription: A good skill\n---\nGood content"))
	h.Upsert("bad/SKILL.md", []byte("---\nname: bad\n---\nmissing description"))
	h.Upsert("also-good/SKILL.md", []byte("---\nname: also-good\ndescription: Another good skill\ntags:\n  - test\n---\nMore content"))

	skills, errs := LoadAll(h)

	if len(skills) != 2 {
		t.Fatalf("expected 2 valid skills, got %d", len(skills))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "bad") {
		t.Errorf("expected error about bad, got %q", errs[0])
	}
}

func TestLoadAllSkipsDirsWithoutSkillMD(t *testing.T) {
	h := home.NewMem()

	h.Upsert("valid/SKILL.md", []byte("---\nname: valid\ndescription: Valid skill\n---\nContent"))
	h.Upsert("noskill/other.txt", []byte("not a skill"))

	skills, errs := LoadAll(h)

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
	if skills[0].Name != "valid" {
		t.Errorf("expected skill name 'valid', got %q", skills[0].Name)
	}
}

func TestLoadAllEmpty(t *testing.T) {
	h := home.NewMem()
	skills, errs := LoadAll(h)
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestLoad(t *testing.T) {
	h := home.NewMem()
	h.Upsert("my-skill/SKILL.md", []byte("---\nname: my-skill\ndescription: Test skill\n---\nContent here"))

	s, err := Load(h, "my-skill")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.Name != "my-skill" {
		t.Errorf("expected name my-skill, got %q", s.Name)
	}
}

func TestLoadNotFound(t *testing.T) {
	h := home.NewMem()
	_, err := Load(h, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}
