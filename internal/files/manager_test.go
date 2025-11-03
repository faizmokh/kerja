package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonthPath(t *testing.T) {
	tmp := t.TempDir()

	mgr, err := NewManager(tmp)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	date := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC)
	path := mgr.MonthPath(date)

	want := filepath.Join(tmp, "2025", "2025-11.md")
	if path != want {
		t.Fatalf("MonthPath() = %q, want %q", path, want)
	}
}

func TestEnsureMonthFileCreatesSkeleton(t *testing.T) {
	tmp := t.TempDir()

	mgr, err := NewManager(tmp)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	date := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected directory %q to exist: %v", dir, err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	wantHeader := "# November 2025\n\n"
	if string(contents) != wantHeader {
		t.Fatalf("month file contents = %q, want %q", contents, wantHeader)
	}

	// Second ensure should not duplicate the header.
	if _, err := mgr.EnsureMonthFile(date); err != nil {
		t.Fatalf("EnsureMonthFile second call: %v", err)
	}
	contentsAgain, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile second: %v", err)
	}
	if string(contentsAgain) != wantHeader {
		t.Fatalf("month file contents after second ensure = %q, want %q", contentsAgain, wantHeader)
	}
}
