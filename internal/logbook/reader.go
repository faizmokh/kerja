package logbook

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/faizmokh/kerja/internal/files"
)

// Reader provides helpers to load sections from Markdown log files.
type Reader struct {
	manager *files.Manager
}

// NewReader wires a reader using the shared files.Manager.
func NewReader(manager *files.Manager) *Reader {
	return &Reader{manager: manager}
}

// Section returns the DateSection for the provided date.
func (r *Reader) Section(ctx context.Context, date time.Time) (DateSection, error) {
	if r == nil || r.manager == nil {
		return DateSection{}, errors.New("reader not initialized with file manager")
	}

	path, err := r.manager.EnsureMonthFile(date)
	if err != nil {
		return DateSection{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return DateSection{}, err
	}
	defer file.Close()

	parser := NewParser(file)
	for {
		section, err := parser.NextSection()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return DateSection{}, ErrSectionNotFound
			}
			return DateSection{}, err
		}
		if section != nil && sameDay(section.Date, date) {
			return *section, nil
		}
	}
}

// SectionsBetween returns all DateSections that exist between the provided
// start and end dates (inclusive). Missing sections are skipped silently.
func (r *Reader) SectionsBetween(ctx context.Context, start, end time.Time) ([]DateSection, error) {
	if r == nil || r.manager == nil {
		return nil, errors.New("reader not initialized with file manager")
	}
	if end.Before(start) {
		return nil, nil
	}

	var sections []DateSection
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		section, err := r.Section(ctx, current)
		if err != nil {
			if errors.Is(err, ErrSectionNotFound) {
				continue
			}
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
