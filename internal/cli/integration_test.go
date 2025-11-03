package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func TestCLIWorkflowEndToEnd(t *testing.T) {
	ctx := context.Background()
	mgr := newTempManager(t)

	date := "2025-11-21"

	// 1. Add a todo entry.
	todoOut := executeCommand(t, newTodoCommand(ctx, mgr),
		"--date", date,
		"--time", "08:15",
		"Write", "integration", "tests", "#quality",
	)
	assertContains(t, todoOut, "[todo] 08:15 Write integration tests (#quality)")

	// 2. Add a done entry.
	logOut := executeCommand(t, newLogCommand(ctx, mgr),
		"--date", date,
		"--time", "09:30",
		"Ship", "patch", "#release",
	)
	assertContains(t, logOut, "[done] 09:30 Ship patch (#release)")

	// 3. List the day to see both entries.
	listOut := executeCommand(t, newListCommand(ctx, mgr),
		"--date", date,
		"--days", "1",
	)
	assertContains(t, listOut, "2025-11-21")
	assertContains(t, listOut, "Write integration tests")
	assertContains(t, listOut, "Ship patch")

	// 4. Toggle the todo to done.
	toggleOut := executeCommand(t, newToggleCommand(ctx, mgr),
		"--date", date,
		"1",
	)
	assertContains(t, toggleOut, "Toggled entry 1: [done]")

	// 5. Edit the second entry's text, time, status, and tags.
	editOut := executeCommand(t, newEditCommand(ctx, mgr),
		"--date", date,
		"--time", "10:45",
		"--status", "todo",
		"2",
		"Update", "patch", "notes", "#release", "#followup",
	)
	assertContains(t, editOut, "[todo] 10:45 Update patch notes (#release, #followup)")

	// 6. Search for tag results within the month.
	searchOut := executeCommand(t, newSearchCommand(ctx, mgr),
		"#release",
		"--date", date,
	)
	assertContains(t, searchOut, `Results for "#release"`)
	assertNotContains(t, searchOut, "#1 [done] 08:15 Write integration tests")
	assertContains(t, searchOut, "2025-11-21 #2 [todo] 10:45 Update patch notes (#release, #followup)")

	// 7. Delete the first entry.
	deleteOut := executeCommand(t, newDeleteCommand(ctx, mgr),
		"--date", date,
		"1",
	)
	assertContains(t, deleteOut, "Deleted entry 1")

	// 8. Confirm the section now only has the edited entry.
	reader := logbook.NewReader(mgr)
	section, err := reader.Section(ctx, mustParseDate(t, date))
	if err != nil {
		t.Fatalf("reader.Section: %v", err)
	}
	if len(section.Entries) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(section.Entries))
	}
	last := section.Entries[0]
	if last.Text != "Update patch notes" || last.Status != logbook.StatusTodo {
		t.Fatalf("unexpected remaining entry: %#v", last)
	}
}

func executeCommand(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute(%q): %v\n%s", args, err, buf.String())
	}
	return buf.String()
}

func assertContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Fatalf("output %q missing substring %q", output, want)
	}
}

func assertNotContains(t *testing.T, output, want string) {
	t.Helper()
	if strings.Contains(output, want) {
		t.Fatalf("output %q unexpectedly contained substring %q", output, want)
	}
}

func newTempManager(t *testing.T) *files.Manager {
	t.Helper()
	base := t.TempDir()
	mgr, err := files.NewManager(base)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

func mustParseDate(t *testing.T, value string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		t.Fatalf("time.ParseInLocation: %v", err)
	}
	return d
}
