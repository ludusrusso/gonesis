package home

import (
	"path/filepath"
	"sort"
	"strings"
)

// MemHome is an in-memory Home for testing.
type MemHome struct {
	Files map[string][]byte
}

// NewMem returns a MemHome with an empty file map.
func NewMem() *MemHome {
	return &MemHome{Files: make(map[string][]byte)}
}

func (m *MemHome) Get(name string) ([]byte, error) {
	data, ok := m.Files[name]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

func (m *MemHome) Search(pattern string) ([]string, error) {
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

func (m *MemHome) Upsert(name string, data []byte) error {
	m.Files[name] = data
	return nil
}

func (m *MemHome) Delete(name string) error {
	if _, ok := m.Files[name]; !ok {
		return ErrNotFound
	}
	delete(m.Files, name)
	return nil
}

func (m *MemHome) Sub(name string) (Home, error) {
	return &subMemHome{parent: m, prefix: name + "/"}, nil
}

func (m *MemHome) ListDirs() ([]string, error) {
	seen := make(map[string]bool)
	var dirs []string
	for name := range m.Files {
		if i := strings.Index(name, "/"); i > 0 {
			dir := name[:i]
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func (m *MemHome) DeleteDir(name string) error {
	prefix := name + "/"
	found := false
	for k := range m.Files {
		if strings.HasPrefix(k, prefix) {
			delete(m.Files, k)
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

// subMemHome is a MemHome scoped to a prefix (subdirectory).
type subMemHome struct {
	parent *MemHome
	prefix string
}

func (s *subMemHome) Get(name string) ([]byte, error) {
	return s.parent.Get(s.prefix + name)
}

func (s *subMemHome) Search(pattern string) ([]string, error) {
	all, err := s.parent.Search(s.prefix + pattern)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(all))
	for i, name := range all {
		result[i] = strings.TrimPrefix(name, s.prefix)
	}
	return result, nil
}

func (s *subMemHome) Upsert(name string, data []byte) error {
	return s.parent.Upsert(s.prefix+name, data)
}

func (s *subMemHome) Delete(name string) error {
	return s.parent.Delete(s.prefix + name)
}

func (s *subMemHome) Sub(name string) (Home, error) {
	return &subMemHome{parent: s.parent, prefix: s.prefix + name + "/"}, nil
}

func (s *subMemHome) ListDirs() ([]string, error) {
	seen := make(map[string]bool)
	var dirs []string
	for name := range s.parent.Files {
		if !strings.HasPrefix(name, s.prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, s.prefix)
		if i := strings.Index(rest, "/"); i > 0 {
			dir := rest[:i]
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func (s *subMemHome) DeleteDir(name string) error {
	prefix := s.prefix + name + "/"
	found := false
	for k := range s.parent.Files {
		if strings.HasPrefix(k, prefix) {
			delete(s.parent.Files, k)
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	return nil
}
