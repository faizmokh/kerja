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

func TestWriterAppendCreatesSection(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC)
	entry := Entry{
		Status: StatusTodo,
		Time:   time.Date(2025, time.November, 2, 9, 45, 0, 0, time.UTC),
		Text:   "Investigate cache invalidation",
		Tags:   []string{"infra", "bug"},
	}

	if err := writer.Append(context.Background(), date, entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	path := mgr.MonthPath(date)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	want := strings.TrimLeft(`
# November 2025

## 2025-11-02
- [ ] [09:45] Investigate cache invalidation #infra #bug
`, "\n")

	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func TestWriterAppendExtendsExistingSection(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 3, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	initial := strings.TrimLeft(`
# November 2025

## 2025-11-03
- [ ] [08:00] Existing todo #ops
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entry := Entry{
		Status: StatusDone,
		Time:   time.Date(2025, time.November, 3, 14, 10, 0, 0, time.UTC),
		Text:   "Deploy fix to production",
		Tags:   []string{"release"},
	}

	if err := writer.Append(context.Background(), date, entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	want := strings.TrimLeft(`
# November 2025

## 2025-11-03
- [ ] [08:00] Existing todo #ops
- [x] [14:10] Deploy fix to production #release
`, "\n")
	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func TestWriterToggleFlipsStatus(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 4, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	initial := strings.TrimLeft(`
# November 2025

## 2025-11-04
- [ ] [09:00] Write unit tests #testing
- [ ] [10:30] Update docs #docs
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	updated, err := writer.Toggle(context.Background(), date, 2)
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if updated.Status != StatusDone {
		t.Fatalf("Toggle returned status = %v, want StatusDone", updated.Status)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := strings.TrimLeft(`
# November 2025

## 2025-11-04
- [ ] [09:00] Write unit tests #testing
- [x] [10:30] Update docs #docs
`, "\n")
	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func TestWriterEditReplacesEntry(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 5, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	initial := strings.TrimLeft(`
# November 2025

## 2025-11-05
- [ ] [13:00] Draft ADR #architecture
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	updatedEntry := Entry{
		Status: StatusDone,
		Time:   time.Date(2025, time.November, 5, 15, 30, 0, 0, time.UTC),
		Text:   "Publish ADR after review",
		Tags:   []string{"architecture", "review"},
	}

	if err := writer.Edit(context.Background(), date, 1, updatedEntry); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := strings.TrimLeft(`
# November 2025

## 2025-11-05
- [x] [15:30] Publish ADR after review #architecture #review
`, "\n")
	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func TestWriterDeleteRemovesEntry(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 6, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	initial := strings.TrimLeft(`
# November 2025

## 2025-11-06
- [ ] [08:30] Review incident report #ops
- [x] [11:45] Plan retro #team
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	deleted, err := writer.Delete(context.Background(), date, 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.Text != "Review incident report" {
		t.Fatalf("Delete returned entry text = %q, want %q", deleted.Text, "Review incident report")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := strings.TrimLeft(`
# November 2025

## 2025-11-06
- [x] [11:45] Plan retro #team
`, "\n")
	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func TestWriterErrorsWhenSectionMissing(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 7, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	// The file has no section for the target date.
	initial := strings.TrimLeft(`
# November 2025

## 2025-11-06
- [x] [11:45] Plan retro #team
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := writer.Toggle(context.Background(), date, 1); !errors.Is(err, ErrSectionNotFound) {
		t.Fatalf("Toggle error = %v, want ErrSectionNotFound", err)
	}
}

func TestWriterErrorsWhenIndexOutOfRange(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := NewWriter(mgr)

	date := time.Date(2025, time.November, 8, 0, 0, 0, 0, time.UTC)
	path, err := mgr.EnsureMonthFile(date)
	if err != nil {
		t.Fatalf("EnsureMonthFile: %v", err)
	}

	initial := strings.TrimLeft(`
# November 2025

## 2025-11-08
- [ ] [09:00] Standup #team
`, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := writer.Delete(context.Background(), date, 2); !errors.Is(err, ErrInvalidIndex) {
		t.Fatalf("Delete error = %v, want ErrInvalidIndex", err)
	}
}
