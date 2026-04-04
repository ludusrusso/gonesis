package home

import (
	"fmt"
	"os"
	"path/filepath"
)

// FSHome is a Home backed by a filesystem directory.
type FSHome struct {
	dir string
}

// New creates a FSHome rooted at dir, creating the directory if needed.
func New(dir string) (*FSHome, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("home: create dir: %w", err)
	}
	return &FSHome{dir: dir}, nil
}

func (h *FSHome) Get(name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(h.dir, name))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return data, err
}

func (h *FSHome) Search(pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(h.dir, pattern))
	if err != nil {
		return nil, err
	}
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = filepath.Base(m)
	}
	return names, nil
}

func (h *FSHome) Upsert(name string, data []byte) error {
	path := filepath.Join(h.dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (h *FSHome) Delete(name string) error {
	err := os.Remove(filepath.Join(h.dir, name))
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

func (h *FSHome) Sub(name string) (Home, error) {
	return New(filepath.Join(h.dir, name))
}

func (h *FSHome) ListDirs() ([]string, error) {
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

func (h *FSHome) DeleteDir(name string) error {
	path := filepath.Join(h.dir, name)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", name)
	}
	return os.RemoveAll(path)
}
