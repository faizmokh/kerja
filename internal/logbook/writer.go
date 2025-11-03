package logbook

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/faizmokh/kerja/internal/files"
)

// Writer handles append, toggle, edit, and delete operations on Markdown log files.
type Writer struct {
	manager *files.Manager
}

// NewWriter wires the dependencies required to manipulate Markdown log files.
func NewWriter(manager *files.Manager) *Writer {
	return &Writer{manager: manager}
}

// Append adds a new entry at the end of the target section, creating the section if needed.
func (w *Writer) Append(ctx context.Context, date time.Time, entry Entry) error {
	if w == nil || w.manager == nil {
		return fmt.Errorf("writer not initialized with file manager")
	}

	entry = normalizeEntryTime(date, entry)

	path, lines, state, err := w.loadSection(ctx, date)
	if err != nil {
		return err
	}

	if state == nil {
		heading := dateHeading(date)
		if needsSeparation(lines) {
			lines = append(lines, "")
		}
		lines = append(lines, heading)
		lines = append(lines, formatEntry(entry))
	} else {
		insertAt := state.end
		lines = insertLine(lines, insertAt, formatEntry(entry))
	}

	return writeLines(path, lines)
}

// Toggle flips StatusTodo <-> StatusDone for the entry at index (1-based) within the section.
func (w *Writer) Toggle(ctx context.Context, date time.Time, index int) (Entry, error) {
	path, lines, state, err := w.loadSection(ctx, date)
	if err != nil {
		return Entry{}, err
	}
	if state == nil {
		return Entry{}, ErrSectionNotFound
	}
	if index < 1 || index > len(state.entryIndexes) {
		return Entry{}, ErrInvalidIndex
	}

	lineIdx := state.entryIndexes[index-1]
	entry := state.section.Entries[index-1]
	switch entry.Status {
	case StatusTodo:
		entry.Status = StatusDone
	case StatusDone:
		entry.Status = StatusTodo
	default:
		entry.Status = StatusTodo
	}

	lines[lineIdx] = formatEntry(entry)
	if err := writeLines(path, lines); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// Edit replaces the entry at index (1-based) with the supplied entry.
func (w *Writer) Edit(ctx context.Context, date time.Time, index int, updated Entry) error {
	updated = normalizeEntryTime(date, updated)

	path, lines, state, err := w.loadSection(ctx, date)
	if err != nil {
		return err
	}
	if state == nil {
		return ErrSectionNotFound
	}
	if index < 1 || index > len(state.entryIndexes) {
		return ErrInvalidIndex
	}

	lineIdx := state.entryIndexes[index-1]
	lines[lineIdx] = formatEntry(updated)
	return writeLines(path, lines)
}

// Delete removes the entry at index (1-based) from the section.
func (w *Writer) Delete(ctx context.Context, date time.Time, index int) (Entry, error) {
	path, lines, state, err := w.loadSection(ctx, date)
	if err != nil {
		return Entry{}, err
	}
	if state == nil {
		return Entry{}, ErrSectionNotFound
	}
	if index < 1 || index > len(state.entryIndexes) {
		return Entry{}, ErrInvalidIndex
	}

	lineIdx := state.entryIndexes[index-1]
	entry := state.section.Entries[index-1]

	lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
	return entry, writeLines(path, lines)
}

// loadSection pulls the current entries for the date to aid writer operations. Implementation pending.
func (w *Writer) loadSection(ctx context.Context, date time.Time) (string, []string, *sectionState, error) {
	if w == nil || w.manager == nil {
		return "", nil, nil, fmt.Errorf("writer not initialized with file manager")
	}

	path, err := w.manager.EnsureMonthFile(date)
	if err != nil {
		return "", nil, nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, nil, err
	}

	lines := splitLines(string(data))
	heading := dateHeading(date)

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i
			break
		}
	}

	if start == -1 {
		return path, lines, nil, nil
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}

	var (
		entryIndexes []int
		entries      []Entry
	)
	sectionDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	for i := start + 1; i < end; i++ {
		line := strings.TrimSpace(lines[i])
		if entry, ok := parseEntryLine(line, sectionDate); ok {
			entryIndexes = append(entryIndexes, i)
			entries = append(entries, entry)
		}
	}

	state := &sectionState{
		section: DateSection{
			Date:    sectionDate,
			Entries: entries,
		},
		start:        start,
		end:          end,
		entryIndexes: entryIndexes,
	}

	return path, lines, state, nil
}

type sectionState struct {
	section      DateSection
	start        int
	end          int
	entryIndexes []int
}

func dateHeading(date time.Time) string {
	return fmt.Sprintf("## %04d-%02d-%02d", date.Year(), date.Month(), date.Day())
}

func splitLines(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(input, "\n")
	// Remove the trailing empty element produced by Split when the input ends with a newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func needsSeparation(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	return strings.TrimSpace(lines[len(lines)-1]) != ""
}

func insertLine(lines []string, index int, line string) []string {
	if index < 0 || index > len(lines) {
		return append(lines, line)
	}
	lines = append(lines[:index], append([]string{line}, lines[index:]...)...)
	return lines
}

func writeLines(path string, lines []string) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "kerja-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if _, err := temp.WriteString(content); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err == nil {
		if err := os.Chmod(temp.Name(), info.Mode()); err != nil {
			return err
		}
	}

	return os.Rename(temp.Name(), path)
}

func formatEntry(entry Entry) string {
	status := ' '
	if entry.Status == StatusDone {
		status = 'x'
	}

	var builder strings.Builder
	builder.Grow(32 + len(entry.Text) + len(entry.Tags)*6)
	fmt.Fprintf(&builder, "- [%c] [%s]", status, entry.Time.Format("15:04"))
	if entry.Text != "" {
		builder.WriteByte(' ')
		builder.WriteString(entry.Text)
	}
	for _, tag := range entry.Tags {
		builder.WriteByte(' ')
		builder.WriteByte('#')
		builder.WriteString(tag)
	}
	return builder.String()
}

func normalizeEntryTime(date time.Time, entry Entry) Entry {
	loc := date.Location()
	if loc == nil {
		loc = time.UTC
	}
	hour := entry.Time.Hour()
	min := entry.Time.Minute()
	if entry.Time.IsZero() {
		hour = 0
		min = 0
	}
	entry.Time = time.Date(date.Year(), date.Month(), date.Day(), hour, min, 0, 0, loc)
	return entry
}
