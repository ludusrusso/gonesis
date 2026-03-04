package homer

import (
	"fmt"
	"os"
	"path/filepath"
)

// FSHomer is a Homer backed by a filesystem directory.
type FSHomer struct {
	dir string
}

// New creates a FSHomer rooted at dir, creating the directory if needed.
func New(dir string) (*FSHomer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("homer: create dir: %w", err)
	}
	return &FSHomer{dir: dir}, nil
}

func (h *FSHomer) Get(name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(h.dir, name))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return data, err
}

func (h *FSHomer) Search(pattern string) ([]string, error) {
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

func (h *FSHomer) Upsert(name string, data []byte) error {
	return os.WriteFile(filepath.Join(h.dir, name), data, 0o644)
}
