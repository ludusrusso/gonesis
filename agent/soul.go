package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// soulPath returns the path to .gonesis/SOUL.md relative to baseDir.
func soulPath(baseDir string) string {
	return filepath.Join(baseDir, ".gonesis", "SOUL.md")
}

// LoadSoul reads SOUL.md from baseDir. Returns (content, exists, err).
func LoadSoul(baseDir string) (string, bool, error) {
	data, err := os.ReadFile(soulPath(baseDir))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading SOUL.md: %w", err)
	}
	return string(data), true, nil
}

// writeSoul creates .gonesis/ dir if needed and writes SOUL.md.
func writeSoul(baseDir, content string) error {
	dir := filepath.Join(baseDir, ".gonesis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating .gonesis dir: %w", err)
	}
	if err := os.WriteFile(soulPath(baseDir), []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing SOUL.md: %w", err)
	}
	return nil
}

// loadWorkspaceFile reads a file from .gonesis/ under baseDir.
// Returns "" if the file does not exist.
func loadWorkspaceFile(baseDir, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(baseDir, ".gonesis", filename))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", filename, err)
	}
	return string(data), nil
}

// BuildSystemPrompt assembles the full system prompt from the embedded agent
// prompt, the runtime soul content, and an optional USER.md file.
func BuildSystemPrompt(baseDir, soulContent string) string {
	sections := []string{
		fmt.Sprintf("# Agent\n\n%s", strings.TrimSpace(agentPrompt)),
	}

	if s := strings.TrimSpace(soulContent); s != "" {
		sections = append(sections, fmt.Sprintf("# Agent Soul\n\n%s", s))
	}

	if userPrefs, err := loadWorkspaceFile(baseDir, "USER.md"); err == nil && strings.TrimSpace(userPrefs) != "" {
		sections = append(sections, fmt.Sprintf("# User Preferences\n\n%s", strings.TrimSpace(userPrefs)))
	}

	return strings.Join(sections, "\n\n")
}
