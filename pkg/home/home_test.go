package home

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/x/file"
)

func TestNew(t *testing.T) {
	t.Run("creates directory and Dir returns path", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "subdir")

		h, err := New(dir)
		if err != nil {
			t.Fatalf("New(%q): %v", dir, err)
		}

		if h.Dir() != dir {
			t.Errorf("Dir() = %q, want %q", h.Dir(), dir)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("path exists but is not a directory")
		}
	})
}

func TestNewTmpHome(t *testing.T) {
	h := NewTmpHome(t)
	if h.Dir() == "" {
		t.Fatal("Dir() is empty")
	}
	info, err := os.Stat(h.Dir())
	if err != nil {
		t.Fatalf("directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("path exists but is not a directory")
	}
}

func TestIdentityFiles(t *testing.T) {
	t.Run("return empty when missing", func(t *testing.T) {
		h := NewTmpHome(t)

		for _, tc := range []struct {
			name string
			f    file.File
		}{
			{"Soul", h.Soul()},
			{"Memory", h.Memory()},
			{"User", h.User()},
		} {
			t.Run(tc.name, func(t *testing.T) {
				content, err := tc.f.Get()
				if err != nil {
					t.Fatalf("Get() error: %v", err)
				}
				if content != "" {
					t.Errorf("Get() = %q, want empty string", content)
				}
			})
		}
	})

	t.Run("round trip", func(t *testing.T) {
		h := NewTmpHome(t)

		for _, tc := range []struct {
			name string
			f    file.File
		}{
			{"Soul", h.Soul()},
			{"Memory", h.Memory()},
			{"User", h.User()},
		} {
			t.Run(tc.name, func(t *testing.T) {
				want := "hello from " + tc.name
				if err := tc.f.Write(want); err != nil {
					t.Fatalf("Write() error: %v", err)
				}
				got, err := tc.f.Get()
				if err != nil {
					t.Fatalf("Get() error: %v", err)
				}
				if got != want {
					t.Errorf("Get() = %q, want %q", got, want)
				}
			})
		}
	})
}

func TestDirectoryAccessors(t *testing.T) {
	h := NewTmpHome(t)
	base := h.Dir()

	for _, tc := range []struct {
		name   string
		got    string
		suffix string
	}{
		{"SkillsDir", h.SkillsDir(), "skills"},
		{"CronsDir", h.CronsDir(), "crons"},
		{"CronResultsDir", h.CronResultsDir(), "cron-results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := filepath.Join(base, tc.suffix)
			if tc.got != want {
				t.Errorf("%s() = %q, want %q", tc.name, tc.got, want)
			}
			if !strings.HasPrefix(tc.got, base) {
				t.Errorf("%s() not under home dir", tc.name)
			}
		})
	}
}
