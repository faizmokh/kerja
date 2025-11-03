package logbook

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestParserParsesSectionsSequentially(t *testing.T) {
	input := `# November 2025

## 2025-11-02
- [x] [09:45] Fixed loan summary layout #ui #bug
- [ ] [11:10] Review PR for lending dashboard #review

## 2025-11-03
- [ ] [10:00] Investigated caching strategy #lending #research
- [x] [14:20] Wrote integration tests #testing
`

	p := NewParser(strings.NewReader(input))

	section, err := p.NextSection()
	if err != nil {
		t.Fatalf("NextSection first call: %v", err)
	}
	wantDate := time.Date(2025, time.November, 2, 0, 0, 0, 0, time.UTC)
	if !section.Date.Equal(wantDate) {
		t.Fatalf("section.Date = %s, want %s", section.Date, wantDate)
	}
	if len(section.Entries) != 2 {
		t.Fatalf("section.Entries length = %d, want 2", len(section.Entries))
	}

	first := section.Entries[0]
	if first.Status != StatusDone {
		t.Fatalf("first.Status = %v, want StatusDone", first.Status)
	}
	wantTime := time.Date(2025, time.November, 2, 9, 45, 0, 0, time.UTC)
	if !first.Time.Equal(wantTime) {
		t.Fatalf("first.Time = %s, want %s", first.Time, wantTime)
	}
	if first.Text != "Fixed loan summary layout" {
		t.Fatalf("first.Text = %q, want %q", first.Text, "Fixed loan summary layout")
	}
	if len(first.Tags) != 2 || first.Tags[0] != "ui" || first.Tags[1] != "bug" {
		t.Fatalf("first.Tags = %#v, want [ui bug]", first.Tags)
	}

	second := section.Entries[1]
	if second.Status != StatusTodo {
		t.Fatalf("second.Status = %v, want StatusTodo", second.Status)
	}
	secondTime := time.Date(2025, time.November, 2, 11, 10, 0, 0, time.UTC)
	if !second.Time.Equal(secondTime) {
		t.Fatalf("second.Time = %s, want %s", second.Time, secondTime)
	}
	if second.Tags[0] != "review" {
		t.Fatalf("second.Tags[0] = %q, want %q", second.Tags[0], "review")
	}

	next, err := p.NextSection()
	if err != nil {
		t.Fatalf("NextSection second call: %v", err)
	}
	wantNextDate := time.Date(2025, time.November, 3, 0, 0, 0, 0, time.UTC)
	if !next.Date.Equal(wantNextDate) {
		t.Fatalf("next.Date = %s, want %s", next.Date, wantNextDate)
	}
	if len(next.Entries) != 2 {
		t.Fatalf("next.Entries length = %d, want 2", len(next.Entries))
	}
	last := next.Entries[1]
	if last.Status != StatusDone {
		t.Fatalf("last.Status = %v, want StatusDone", last.Status)
	}
	if last.Text != "Wrote integration tests" {
		t.Fatalf("last.Text = %q, want %q", last.Text, "Wrote integration tests")
	}
	if len(last.Tags) != 1 || last.Tags[0] != "testing" {
		t.Fatalf("last.Tags = %#v, want [testing]", last.Tags)
	}

	if _, err := p.NextSection(); !errors.Is(err, io.EOF) {
		t.Fatalf("NextSection after EOF error = %v, want io.EOF", err)
	}
}

func TestParserSkipsInvalidLinesAndHeadings(t *testing.T) {
	input := `# November 2025

## not-a-date
- [ ] [12:00] This line is ignored because section invalid

## 2025-11-02
Some stray text
- [x] [08:15] Valid entry #shipping #infra
### Notes
- [ ] Missing brackets and time

## 2025-11-03
- [x] [09:00] Another valid entry #ui
# Extra heading
- [ ] [10:00] Trailing tags #tag1 #tag2
`

	p := NewParser(strings.NewReader(input))

	first, err := p.NextSection()
	if err != nil {
		t.Fatalf("NextSection first call: %v", err)
	}
	if len(first.Entries) != 1 {
		t.Fatalf("first section entries = %d, want 1", len(first.Entries))
	}
	if first.Entries[0].Text != "Valid entry" {
		t.Fatalf("first entry text = %q, want %q", first.Entries[0].Text, "Valid entry")
	}
	if len(first.Entries[0].Tags) != 2 {
		t.Fatalf("first entry tags length = %d, want 2", len(first.Entries[0].Tags))
	}

	second, err := p.NextSection()
	if err != nil {
		t.Fatalf("NextSection second call: %v", err)
	}
	if len(second.Entries) != 2 {
		t.Fatalf("second section entries = %d, want 2", len(second.Entries))
	}
	if second.Entries[0].Text != "Another valid entry" {
		t.Fatalf("second[0] text = %q, want %q", second.Entries[0].Text, "Another valid entry")
	}
	if second.Entries[1].Status != StatusTodo {
		t.Fatalf("second[1].Status = %v, want StatusTodo", second.Entries[1].Status)
	}
	if _, err := p.NextSection(); !errors.Is(err, io.EOF) {
		t.Fatalf("NextSection third call error = %v, want io.EOF", err)
	}
}

func TestParserWithNilReader(t *testing.T) {
	p := NewParser(nil)
	if _, err := p.NextSection(); !errors.Is(err, io.EOF) {
		t.Fatalf("NextSection with nil reader error = %v, want io.EOF", err)
	}
}
