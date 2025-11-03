package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func TestTodayCommandPrintsEntries(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	writer := logbook.NewWriter(mgr)
	date := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.Local)

	if err := writer.Append(context.Background(), date, logbook.Entry{
		Status: logbook.StatusDone,
		Time:   time.Date(2025, time.November, 2, 9, 45, 0, 0, time.Local),
		Text:   "Fixed loan summary layout",
		Tags:   []string{"ui", "bug"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := writer.Append(context.Background(), date, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, time.November, 2, 11, 10, 0, 0, time.Local),
		Text:   "Review PR for lending dashboard",
		Tags:   []string{"review"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newTodayCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-02"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2025-11-02") {
		t.Fatalf("output missing date: %q", output)
	}
	if !strings.Contains(output, "[done] 09:45 Fixed loan summary layout (#ui, #bug)") {
		t.Fatalf("output missing first entry: %q", output)
	}
	if !strings.Contains(output, "[todo] 11:10 Review PR for lending dashboard (#review)") {
		t.Fatalf("output missing second entry: %q", output)
	}
}

func TestTodayCommandWithoutEntries(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	cmd := newTodayCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-03"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No entries for 2025-11-03") {
		t.Fatalf("unexpected output: %q", output)
	}
}
