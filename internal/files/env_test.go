package files

import (
	"path/filepath"
	"testing"
)

func TestResolveBasePathHonorsKerjaHome(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom-root")

	t.Setenv("KERJA_HOME", custom)

	got, err := ResolveBasePath()
	if err != nil {
		t.Fatalf("ResolveBasePath() error = %v", err)
	}
	if got != custom {
		t.Fatalf("ResolveBasePath() = %q, want %q", got, custom)
	}
}

func TestResolveBasePathExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KERJA_HOME", "~/kerja-data")

	got, err := ResolveBasePath()
	if err != nil {
		t.Fatalf("ResolveBasePath() error = %v", err)
	}

	want := filepath.Join(home, "kerja-data")
	if got != want {
		t.Fatalf("ResolveBasePath() = %q, want %q", got, want)
	}
}

func TestResolveBasePathDefaultsToHomeDotKerja(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KERJA_HOME", "")

	got, err := ResolveBasePath()
	if err != nil {
		t.Fatalf("ResolveBasePath() error = %v", err)
	}

	want := filepath.Join(home, DefaultDirName)
	if got != want {
		t.Fatalf("ResolveBasePath() = %q, want %q", got, want)
	}
}
