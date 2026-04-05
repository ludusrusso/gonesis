package file

import (
	"path/filepath"
	"testing"
)

func TestFSFile(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		dir := t.TempDir()
		f := NewFSFile(filepath.Join(dir, "test.txt"))
		RunFileSpec(t, f)
	})

	t.Run("ReplaceMissing", func(t *testing.T) {
		dir := t.TempDir()
		f := NewFSFile(filepath.Join(dir, "nonexistent.txt"))
		RunFileSpecReplaceMissing(t, f)
	})
}
