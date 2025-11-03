package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func TestPrevCommandShowsPreviousDay(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	prevDate := time.Date(2025, 11, 1, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), prevDate, logbook.Entry{
		Status: logbook.StatusDone,
		Time:   time.Date(2025, 11, 1, 9, 0, 0, 0, time.Local),
		Text:   "Previous day task",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newPrevCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-02"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2025-11-01") {
		t.Fatalf("output missing date: %q", output)
	}
	if !strings.Contains(output, "Previous day task") {
		t.Fatalf("output missing entry: %q", output)
	}
}

func TestNextCommandShowsNextDay(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	nextDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), nextDate, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 3, 10, 0, 0, 0, time.Local),
		Text:   "Next day task",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newNextCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-02"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2025-11-03") {
		t.Fatalf("output missing date: %q", output)
	}
	if !strings.Contains(output, "Next day task") {
		t.Fatalf("output missing entry: %q", output)
	}
}

func TestJumpCommandDisplaysRequestedDate(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	target := time.Date(2025, 11, 4, 0, 0, 0, 0, time.Local)
	if err := writer.Append(context.Background(), target, logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 4, 15, 30, 0, 0, time.Local),
		Text:   "Jump target task",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newJumpCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"2025-11-04"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Jump target task") {
		t.Fatalf("output missing entry: %q", output)
	}
}

func TestListCommandAggregatesRange(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	for i := 0; i < 3; i++ {
		day := time.Date(2025, 11, 8+i, 0, 0, 0, 0, time.Local)
		if err := writer.Append(context.Background(), day, logbook.Entry{
			Status: logbook.StatusTodo,
			Time:   time.Date(2025, 11, 8+i, 9, 0, 0, 0, time.Local),
			Text:   fmt.Sprintf("Task %d", i),
		}); err != nil {
			t.Fatalf("Append day %d: %v", i, err)
		}
	}

	cmd := newListCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--date", "2025-11-10", "--days", "3"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2025-11-08") || !strings.Contains(output, "2025-11-10") {
		t.Fatalf("output missing expected dates: %q", output)
	}
	if strings.Count(output, "Task") != 3 {
		t.Fatalf("expected three tasks, output: %q", output)
	}
}

func TestSearchCommandFindsMatches(t *testing.T) {
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	writer := logbook.NewWriter(mgr)

	dates := []time.Time{
		time.Date(2025, 11, 18, 0, 0, 0, 0, time.Local),
		time.Date(2025, 11, 19, 0, 0, 0, 0, time.Local),
	}

	if err := writer.Append(context.Background(), dates[0], logbook.Entry{
		Status: logbook.StatusDone,
		Time:   time.Date(2025, 11, 18, 9, 0, 0, 0, time.Local),
		Text:   "Investigate cache bug",
		Tags:   []string{"infra"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := writer.Append(context.Background(), dates[1], logbook.Entry{
		Status: logbook.StatusTodo,
		Time:   time.Date(2025, 11, 19, 14, 15, 0, 0, time.Local),
		Text:   "Write summary",
		Tags:   []string{"report"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	cmd := newSearchCommand(context.Background(), mgr)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"bug", "--date", "2025-11-20"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Results for \"bug\"") {
		t.Fatalf("output missing header: %q", output)
	}
	if !strings.Contains(output, "2025-11-18 #1") {
		t.Fatalf("output missing match line: %q", output)
	}
	if strings.Contains(output, "2025-11-19") {
		t.Fatalf("output unexpectedly included unmatched entry: %q", output)
	}
}
