package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetFrontmatterField(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		key   string
		value any
		want  string
	}{
		{
			name:  "update existing field preserves order and body",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: false\n---\nDo a thing\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: true\n---\nDo a thing\n",
		},
		{
			name:  "add missing field appends at end of frontmatter",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: true\n---\nbody\n",
		},
		{
			name:  "preserves inline comment on existing field",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: false # set by CI\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: true # set by CI\n---\nbody\n",
		},
		{
			name:  "preserves unrelated fields",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\ntimeout: 30m\ndescription: whatever\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\ntimeout: 30m\ndescription: whatever\nsuspended: true\n---\nbody\n",
		},
		{
			name:  "preserves trailing blank line in frontmatter when appending",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\n\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: true\n\n---\nbody\n",
		},
		{
			name:  "update preserves indentation",
			in:    "---\n  name: foo\n  cron: \"0 9 * * *\"\n  suspended: false\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\n  name: foo\n  cron: \"0 9 * * *\"\n  suspended: true\n---\nbody\n",
		},
		{
			name:  "body with trailing content is untouched",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\n---\nline one\nline two\n",
			key:   "suspended",
			value: false,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: false\n---\nline one\nline two\n",
		},
		{
			name:  "toggle true to false",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: true\n---\nbody\n",
			key:   "suspended",
			value: false,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended: false\n---\nbody\n",
		},
		{
			name:  "keys with similar prefix are not confused",
			in:    "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended_at: 2026-01-01\n---\nbody\n",
			key:   "suspended",
			value: true,
			want:  "---\nname: foo\ncron: \"0 9 * * *\"\nsuspended_at: 2026-01-01\nsuspended: true\n---\nbody\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "job.md")
			if err := os.WriteFile(path, []byte(tc.in), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if err := SetFrontmatterField(path, tc.key, tc.value); err != nil {
				t.Fatalf("SetFrontmatterField: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("content mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}

func TestSetFrontmatterFieldErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		err := SetFrontmatterField(filepath.Join(dir, "no-such-file.md"), "suspended", true)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("no opening delimiter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.md")
		if err := os.WriteFile(path, []byte("name: foo\ncron: \"0 9 * * *\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := SetFrontmatterField(path, "suspended", true); err == nil {
			t.Fatal("expected error for missing frontmatter delimiter")
		}
	})

	t.Run("no closing delimiter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.md")
		if err := os.WriteFile(path, []byte("---\nname: foo\ncron: \"0 9 * * *\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := SetFrontmatterField(path, "suspended", true); err == nil {
			t.Fatal("expected error for missing closing delimiter")
		}
	})
}

func TestSetFrontmatterFieldRoundTripsThroughParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.md")
	original := "---\nname: demo\ncron: \"*/5 * * * *\"\n---\nGenerate a summary.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetFrontmatterField(path, "suspended", true); err != nil {
		t.Fatalf("SetFrontmatterField: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	job, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse after set: %v", err)
	}
	if !job.Suspended {
		t.Errorf("expected Suspended=true after set, got false")
	}
	if job.Name != "demo" || job.Schedule != "*/5 * * * *" || job.Prompt != "Generate a summary." {
		t.Errorf("parsed fields changed unexpectedly: %+v", job)
	}
}
