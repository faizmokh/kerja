package logbook

import "time"

// Entry represents a single Markdown task line within a dated section.
type Entry struct {
	Status Status
	Time   time.Time
	Text   string
	Tags   []string
}

// Status expresses whether an entry is still a todo or already done.
type Status uint8

const (
	// StatusTodo marks entries that still need doing.
	StatusTodo Status = iota
	// StatusDone marks entries that are completed.
	StatusDone
)

// DateSection groups entries beneath the same YYYY-MM-DD heading.
type DateSection struct {
	Date   time.Time
	Entries []Entry
}

