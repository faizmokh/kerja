package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/logbook"
)

func resolveDate(dateFlag string) (time.Time, error) {
	if dateFlag == "" {
		now := time.Now().In(time.Local)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	parsed, err := time.ParseInLocation("2006-01-02", dateFlag, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}
	return parsed, nil
}

func resolveTime(date time.Time, timeFlag string) (time.Time, error) {
	if timeFlag == "" {
		now := time.Now().In(date.Location())
		return time.Date(date.Year(), date.Month(), date.Day(), now.Hour(), now.Minute(), 0, 0, date.Location()), nil
	}

	parsed, err := time.ParseInLocation("15:04", timeFlag, date.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, date.Location()), nil
}

func parseTextAndTags(args []string) (string, []string) {
	var (
		textParts []string
		tags      []string
	)

	for _, arg := range args {
		if strings.HasPrefix(arg, "#") && len(arg) > 1 {
			tags = append(tags, strings.TrimPrefix(arg, "#"))
			continue
		}
		textParts = append(textParts, arg)
	}

	return strings.TrimSpace(strings.Join(textParts, " ")), tags
}

func formatEntry(entry logbook.Entry) string {
	status := "todo"
	if entry.Status == logbook.StatusDone {
		status = "done"
	}

	builder := strings.Builder{}
	builder.Grow(32 + len(entry.Text) + len(entry.Tags)*6)

	builder.WriteString("[")
	builder.WriteString(status)
	builder.WriteString("] ")
	builder.WriteString(entry.Time.Format("15:04"))

	if entry.Text != "" {
		builder.WriteString(" ")
		builder.WriteString(entry.Text)
	}

	if len(entry.Tags) > 0 {
		builder.WriteString(" (")
		for i, tag := range entry.Tags {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString("#")
			builder.WriteString(tag)
		}
		builder.WriteString(")")
	}

	return builder.String()
}

func parseStatusFlag(value string, current logbook.Status) (logbook.Status, error) {
	if value == "" {
		return current, nil
	}

	switch strings.ToLower(value) {
	case "todo":
		return logbook.StatusTodo, nil
	case "done":
		return logbook.StatusDone, nil
	default:
		return current, fmt.Errorf("invalid status %q (expected todo|done)", value)
	}
}

func printMissingSection(cmd *cobra.Command, date time.Time) {
	fmt.Fprintf(cmd.OutOrStdout(), "No entries for %s\n", date.Format("2006-01-02"))
}

func printSection(cmd *cobra.Command, section logbook.DateSection) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s\n", section.Date.Format("2006-01-02"))
	if len(section.Entries) == 0 {
		fmt.Fprintln(out, "(no entries)")
		return nil
	}

	for i, entry := range section.Entries {
		fmt.Fprintf(out, "%d. %s\n", i+1, formatEntry(entry))
	}
	return nil
}

func printSections(cmd *cobra.Command, sections []logbook.DateSection) error {
	if len(sections) == 0 {
		return nil
	}
	for i, section := range sections {
		if err := printSection(cmd, section); err != nil {
			return err
		}
		if i < len(sections)-1 {
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}
	return nil
}
