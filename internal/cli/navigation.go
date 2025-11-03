package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func newPrevCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "prev",
		Short: "Show the previous day's log entries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}
			target := date.AddDate(0, 0, -1)
			reader := logbook.NewReader(manager)
			return displaySection(ctx, cmd, reader, target)
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Reference date in YYYY-MM-DD (default: today)")

	return cmd
}

func newNextCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "next",
		Short: "Show the next day's log entries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}
			target := date.AddDate(0, 0, 1)
			reader := logbook.NewReader(manager)
			return displaySection(ctx, cmd, reader, target)
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Reference date in YYYY-MM-DD (default: today)")

	return cmd
}

func newJumpCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jump <date>",
		Short: "Show entries for the specified date.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := time.ParseInLocation("2006-01-02", args[0], time.Local)
			if err != nil {
				return fmt.Errorf("parse date: %w", err)
			}
			reader := logbook.NewReader(manager)
			return displaySection(ctx, cmd, reader, target)
		},
	}

	return cmd
}

func newListCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var (
		dateFlag string
		daysFlag int
		weekFlag bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List entries across a range of days.",
		RunE: func(cmd *cobra.Command, args []string) error {
			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			days := daysFlag
			if weekFlag {
				days = 7
			}
			if days <= 0 {
				days = 1
			}

			start := date.AddDate(0, 0, -(days - 1))
			reader := logbook.NewReader(manager)
			sections, err := reader.SectionsBetween(ctx, start, date)
			if err != nil {
				return err
			}

			if len(sections) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No entries between %s and %s\n",
					start.Format("2006-01-02"), date.Format("2006-01-02"))
				return nil
			}

			return printSections(cmd, sections)
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "End date in YYYY-MM-DD (default: today)")
	cmd.Flags().IntVar(&daysFlag, "days", 0, "Number of days to include ending on target date")
	cmd.Flags().BoolVar(&weekFlag, "week", false, "Shortcut for --days=7")

	return cmd
}

func newSearchCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Search entries by text or tag within the month.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			term := strings.TrimSpace(args[0])
			if term == "" {
				return fmt.Errorf("term is required")
			}
			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
			endOfMonth := startOfMonth.AddDate(0, 1, -1)

			reader := logbook.NewReader(manager)
			sections, err := reader.SectionsBetween(ctx, startOfMonth, endOfMonth)
			if err != nil {
				return err
			}

			results := filterSectionsByTerm(sections, term)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Results for %q in %s\n", term, startOfMonth.Format("2006-01"))
			if len(results) == 0 {
				fmt.Fprintln(out, "(no matches)")
				return nil
			}

			for _, res := range results {
				fmt.Fprintf(out, "%s #%d %s\n",
					res.section.Date.Format("2006-01-02"),
					res.index+1,
					formatEntry(res.entry),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Reference date in YYYY-MM-DD (default: today)")

	return cmd
}

func displaySection(ctx context.Context, cmd *cobra.Command, reader *logbook.Reader, date time.Time) error {
	section, err := reader.Section(ctx, date)
	if err != nil {
		if errors.Is(err, logbook.ErrSectionNotFound) {
			printMissingSection(cmd, date)
			return nil
		}
		return err
	}
	return printSection(cmd, section)
}

type searchResult struct {
	section logbook.DateSection
	entry   logbook.Entry
	index   int
}

func filterSectionsByTerm(sections []logbook.DateSection, term string) []searchResult {
	var results []searchResult
	needle := strings.ToLower(term)
	tagNeedle := strings.TrimPrefix(needle, "#")
	matchTagOnly := strings.HasPrefix(needle, "#")

	for _, section := range sections {
		for idx, entry := range section.Entries {
			if matchesEntry(entry, needle, tagNeedle, matchTagOnly) {
				results = append(results, searchResult{
					section: section,
					entry:   entry,
					index:   idx,
				})
			}
		}
	}

	return results
}

func matchesEntry(entry logbook.Entry, needle, tagNeedle string, tagOnly bool) bool {
	text := strings.ToLower(entry.Text)
	if !tagOnly && strings.Contains(text, needle) {
		return true
	}

	for _, tag := range entry.Tags {
		tagLower := strings.ToLower(tag)
		if tagOnly {
			if tagLower == tagNeedle {
				return true
			}
			continue
		}
		if strings.Contains(tagLower, needle) || tagLower == tagNeedle {
			return true
		}
	}
	return false
}
