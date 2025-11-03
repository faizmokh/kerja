package logbook

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/faizmokh/kerja/internal/files"
)

func TestReaderSectionReturnsEntries(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	reader := NewReader(mgr)

	date := time.Date(2025, time.November, 9, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	content := strings.TrimLeft(`
# November 2025

## 2025-11-08
- [ ] [10:00] Other day entry #misc

## 2025-11-09
- [x] [09:15] Ship feature flag #release
- [ ] [13:45] Draft follow-up ticket #ops
`, "\n")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	section, err := reader.Section(context.Background(), date)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if len(section.Entries) != 2 {
		t.Fatalf("section entries = %d, want 2", len(section.Entries))
	}
	if section.Entries[0].Status != StatusDone {
		t.Fatalf("first entry status = %v, want StatusDone", section.Entries[0].Status)
	}
	if section.Entries[1].Text != "Draft follow-up ticket" {
		t.Fatalf("second entry text = %q, want %q", section.Entries[1].Text, "Draft follow-up ticket")
	}
}

func TestReaderSectionMissingReturnsError(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	reader := NewReader(mgr)

	date := time.Date(2025, time.November, 10, 0, 0, 0, 0, time.UTC)
	if _, err := reader.Section(context.Background(), date); err == nil {
		t.Fatalf("Section expected error, got nil")
	} else if !errors.Is(err, ErrSectionNotFound) {
		t.Fatalf("Section error = %v, want ErrSectionNotFound", err)
	}
}

func TestReaderSectionsBetween(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	reader := NewReader(mgr)
	writer := NewWriter(mgr)

	first := time.Date(2025, time.November, 11, 0, 0, 0, 0, time.UTC)
	second := time.Date(2025, time.November, 12, 0, 0, 0, 0, time.UTC)

	if err := writer.Append(context.Background(), first, Entry{
		Status: StatusTodo,
		Time:   time.Date(2025, time.November, 11, 9, 0, 0, 0, time.UTC),
		Text:   "First day entry",
	}); err != nil {
		t.Fatalf("Append first: %v", err)
	}
	if err := writer.Append(context.Background(), second, Entry{
		Status: StatusDone,
		Time:   time.Date(2025, time.November, 12, 10, 0, 0, 0, time.UTC),
		Text:   "Second day entry",
	}); err != nil {
		t.Fatalf("Append second: %v", err)
	}

	sections, err := reader.SectionsBetween(context.Background(), first, second.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("SectionsBetween: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("sections len = %d, want 2", len(sections))
	}
	if sections[0].Date.Day() != 11 || len(sections[0].Entries) != 1 {
		t.Fatalf("first section unexpected: %#v", sections[0])
	}
	if sections[1].Entries[0].Status != StatusDone {
		t.Fatalf("second section entry status = %v, want StatusDone", sections[1].Entries[0].Status)
	}
}
