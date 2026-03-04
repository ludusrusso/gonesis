package homer

import (
	"path/filepath"
	"sort"
)

// MemHomer is an in-memory Homer for testing.
type MemHomer struct {
	Files map[string][]byte
}

// NewMem returns a MemHomer with an empty file map.
func NewMem() *MemHomer {
	return &MemHomer{Files: make(map[string][]byte)}
}

func (m *MemHomer) Get(name string) ([]byte, error) {
	data, ok := m.Files[name]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

func (m *MemHomer) Search(pattern string) ([]string, error) {
	var matches []string
	for name := range m.Files {
		ok, err := filepath.Match(pattern, name)
		if err != nil {
			return nil, err
		}
		if ok {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func (m *MemHomer) Upsert(name string, data []byte) error {
	m.Files[name] = data
	return nil
}
