package logbook

import "errors"

// ErrSectionNotFound is returned when the targeted date heading cannot be located.
var ErrSectionNotFound = errors.New("date section not found")

// ErrInvalidIndex indicates the caller referenced an entry index outside the section bounds.
var ErrInvalidIndex = errors.New("entry index out of range")
