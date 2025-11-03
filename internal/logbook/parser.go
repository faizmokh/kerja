package logbook

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrNotImplemented is returned by parser stubs until storage work lands.
	ErrNotImplemented = errors.New("not implemented")
)

// Parser incrementally reads Markdown logbooks and emits sections as they are discovered.
type Parser struct {
	r        io.Reader
	scanner  *bufio.Scanner
	pending  *DateSection
	initDone bool
}

// NewParser returns a parser ready to tokenize Markdown from r.
func NewParser(r io.Reader) *Parser {
	return &Parser{r: r}
}

// NextSection will eventually stream the next parsed DateSection.
func (p *Parser) NextSection() (*DateSection, error) {
	if p.r == nil && p.scanner == nil && p.pending == nil {
		return nil, io.EOF
	}

	if !p.initDone {
		if p.r == nil {
			return nil, io.EOF
		}
		p.scanner = bufio.NewScanner(p.r)
		p.initDone = true
	}

	section := p.pending
	p.pending = nil

	for {
		if section == nil {
			var err error
			section, err = p.consumeUntilSection()
			if err != nil {
				return nil, err
			}
			if section == nil {
				return nil, io.EOF
			}
		}

		for p.scanner.Scan() {
			line := strings.TrimSpace(p.scanner.Text())
			if date, ok := parseSectionHeading(line); ok {
				p.pending = &DateSection{Date: date}
				return section, nil
			}

			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}

			if entry, ok := parseEntryLine(line, section.Date); ok {
				section.Entries = append(section.Entries, entry)
			}
		}

		if err := p.scanner.Err(); err != nil {
			return nil, err
		}

		if section != nil {
			return section, nil
		}
	}
}

func (p *Parser) consumeUntilSection() (*DateSection, error) {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if date, ok := parseSectionHeading(line); ok {
			return &DateSection{Date: date}, nil
		}
	}

	if err := p.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

var entryPattern = regexp.MustCompile(`^- \[( |x)\] \[(\d{2}:\d{2})\] (.*)$`)

func parseEntryLine(line string, date time.Time) (Entry, bool) {
	matches := entryPattern.FindStringSubmatch(line)
	if matches == nil {
		return Entry{}, false
	}

	status := StatusTodo
	if matches[1] == "x" {
		status = StatusDone
	}

	parsedTime, err := time.Parse("15:04", matches[2])
	if err != nil {
		return Entry{}, false
	}

	entryTime := time.Date(
		date.Year(), date.Month(), date.Day(),
		parsedTime.Hour(), parsedTime.Minute(), 0, 0,
		date.Location(),
	)

	text, tags := extractTextAndTags(matches[3])

	return Entry{
		Status: status,
		Time:   entryTime,
		Text:   text,
		Tags:   tags,
	}, true
}

func parseSectionHeading(line string) (time.Time, bool) {
	if !strings.HasPrefix(line, "## ") {
		return time.Time{}, false
	}
	dateStr := strings.TrimSpace(line[3:])
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}

func extractTextAndTags(rest string) (string, []string) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", nil
	}

	var tags []string
	text := rest

	// If tags exist they'll follow a space and start with '#'.
	tagStart := strings.Index(rest, " #")
	if tagStart >= 0 {
		text = strings.TrimSpace(rest[:tagStart])
		tags = parseTags(rest[tagStart+1:])
	} else if strings.HasPrefix(rest, "#") {
		text = ""
		tags = parseTags(rest)
	} else {
		text = strings.TrimSpace(rest)
	}

	return text, tags
}

func parseTags(segment string) []string {
	fields := strings.Fields(segment)
	var tags []string
	for _, field := range fields {
		if strings.HasPrefix(field, "#") && len(field) > 1 {
			tag := strings.TrimLeft(field, "#")
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}
