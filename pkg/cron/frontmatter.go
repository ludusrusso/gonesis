package cron

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SetFrontmatterField rewrites a single YAML frontmatter field in the file at path
// without reserializing the whole document. If the field exists, its value is
// replaced in place, preserving leading indentation and any trailing inline
// comment. If the field is absent, it is appended at the end of the frontmatter
// block. Unrelated fields, field order, body, and surrounding whitespace are
// untouched.
func SetFrontmatterField(path, key string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cron: read %s: %w", path, err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("cron: missing frontmatter delimiter")
	}
	rest := content[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fmt.Errorf("cron: missing closing frontmatter delimiter")
	}

	frontmatter := rest[:idx]
	trailer := rest[idx:] // begins with "\n---"

	scalar, err := marshalScalar(value)
	if err != nil {
		return err
	}

	newFrontmatter := replaceOrAppendField(frontmatter, key, scalar)

	newContent := "---\n" + newFrontmatter + trailer
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil { //nolint:gosec // caller-provided path, writing alongside the file we just read
		return fmt.Errorf("cron: write %s: %w", path, err)
	}
	return nil
}

func replaceOrAppendField(frontmatter, key, scalar string) string {
	lines := strings.Split(frontmatter, "\n")
	keyPrefix := key + ":"

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, keyPrefix) {
			continue
		}
		after := trimmed[len(keyPrefix):]
		if after != "" && after[0] != ' ' && after[0] != '\t' && after[0] != '#' {
			// e.g. "keysuffix:" shares the prefix but is a different key.
			continue
		}
		indent := line[:len(line)-len(trimmed)]
		comment := extractInlineComment(after)
		newLine := indent + key + ": " + scalar
		if comment != "" {
			newLine = newLine + " " + comment
		}
		lines[i] = newLine
		return strings.Join(lines, "\n")
	}

	// Field absent: append at the end of the frontmatter. If the block ends
	// with one or more blank lines, insert before them so any trailing
	// whitespace is preserved.
	insertAt := len(lines)
	for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	lines = slices.Insert(lines, insertAt, key+": "+scalar)
	return strings.Join(lines, "\n")
}

// extractInlineComment returns the trailing " # ..." comment in value (including
// the leading '#'), or "" if none is present. Only whitespace-preceded '#' is
// treated as a comment to avoid mangling quoted scalars that contain '#'.
func extractInlineComment(valueAndMaybeComment string) string {
	for i := 0; i < len(valueAndMaybeComment); i++ {
		c := valueAndMaybeComment[i]
		if c != '#' {
			continue
		}
		if i == 0 || valueAndMaybeComment[i-1] == ' ' || valueAndMaybeComment[i-1] == '\t' {
			return strings.TrimRight(valueAndMaybeComment[i:], " \t")
		}
	}
	return ""
}

// marshalScalar renders value as a single-line YAML scalar.
func marshalScalar(value any) (string, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("cron: marshal value: %w", err)
	}
	return strings.TrimRight(string(data), "\n"), nil
}
