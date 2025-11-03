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

func TestLogCommandAppendsDoneEntry(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	cmd := newLogCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-12", "--time", "09:30", "Ship", "feature", "#release"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	reader := logbook.NewReader(mgr)
	section, err := reader.Section(context.Background(), time.Date(2025, 11, 12, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if len(section.Entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(section.Entries))
	}
	entry := section.Entries[0]
	if entry.Status != logbook.StatusDone {
		t.Fatalf("status = %v, want StatusDone", entry.Status)
	}
	if got := buf.String(); !strings.Contains(got, "[done] 09:30 Ship feature (#release)") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestTodoCommandAppendsTodoEntry(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	cmd := newTodoCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-13", "--time", "14:15", "Write", "tests", "#quality"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	reader := logbook.NewReader(mgr)
	section, err := reader.Section(context.Background(), time.Date(2025, 11, 13, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if len(section.Entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(section.Entries))
	}
	if section.Entries[0].Status != logbook.StatusTodo {
		t.Fatalf("status = %v, want StatusTodo", section.Entries[0].Status)
	}
}

func TestToggleCommandFlipsStatus(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	date := time.Date(2025, 11, 14, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), date, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 14, 9, 0, 0, 0, time.Local),
		Text:   "Todo entry",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newToggleCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-14", "1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	reader := logbook.NewReader(mgr)
	section, err := reader.Section(context.Background(), date)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if section.Entries[0].Status != logbook.StatusDone {
		t.Fatalf("status = %v, want StatusDone", section.Entries[0].Status)
	}
	if got := buf.String(); !strings.Contains(got, "[done]") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestEditCommandUpdatesFields(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	date := time.Date(2025, 11, 15, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), date, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 15, 10, 0, 0, 0, time.Local),
		Text:   "Draft doc",
		Tags:   []string{"docs"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newEditCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-15", "--time", "13:30", "--status", "done", "1", "Finalize", "doc", "#docs", "#review"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	reader := logbook.NewReader(mgr)
	section, err := reader.Section(context.Background(), date)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	entry := section.Entries[0]
	if entry.Status != logbook.StatusDone {
		t.Fatalf("status = %v, want StatusDone", entry.Status)
	}
	if entry.Time.Hour() != 13 || entry.Time.Minute() != 30 {
		t.Fatalf("time = %v, want 13:30", entry.Time.Format("15:04"))
	}
	if entry.Text != "Finalize doc" {
		t.Fatalf("text = %q, want %q", entry.Text, "Finalize doc")
	}
	if len(entry.Tags) != 2 || entry.Tags[1] != "review" {
		t.Fatalf("tags = %#v, want [docs review]", entry.Tags)
	}
	if got := buf.String(); !strings.Contains(got, "[done] 13:30 Finalize doc (#docs, #review)") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestDeleteCommandRemovesEntry(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	date := time.Date(2025, 11, 16, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), date, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 16, 11, 0, 0, 0, time.Local),
		Text:   "Remove entry",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newDeleteCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-16", "1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	reader := logbook.NewReader(mgr)
	section, err := reader.Section(context.Background(), date)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if len(section.Entries) != 0 {
		t.Fatalf("expected no entries after delete, got %d", len(section.Entries))
	}
	if got := buf.String(); !strings.Contains(got, "Deleted entry 1") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestToggleCommandInvalidIndex(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	cmd := newToggleCommand(context.Background(), mgr)
	cmd.SetArgs([]string{"--date", "2025-11-17", "0"})

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for invalid index")
	}
	if !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("unexpected error: %v", err)
	}
}
