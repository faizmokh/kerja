package cli

import (
	"context"
	"encoding/json"
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
	var (
		dateFlag      string
		caseSensitive bool
		outputJSON    bool
		includeText   bool
	)

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

			results := filterSectionsByTerm(sections, term, caseSensitive, includeText)
			if outputJSON {
				return printSearchResultsJSON(cmd, results)
			}
			return printSearchResultsText(cmd, term, startOfMonth, results)
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Reference date in YYYY-MM-DD (default: today)")
	cmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Match term with case sensitivity")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Emit results as JSON objects")
	cmd.Flags().BoolVar(&includeText, "include-text", false, "Include body text when matching tag-only searches")

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

func filterSectionsByTerm(sections []logbook.DateSection, term string, caseSensitive bool, includeText bool) []searchResult {
	var results []searchResult
	needle := term
	tagNeedle := strings.TrimPrefix(term, "#")
	matchTagOnly := strings.HasPrefix(term, "#")

	for _, section := range sections {
		for idx, entry := range section.Entries {
			if matchesEntry(entry, needle, tagNeedle, matchTagOnly, caseSensitive, includeText) {
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

func matchesEntry(entry logbook.Entry, needle, tagNeedle string, tagOnly bool, caseSensitive bool, includeText bool) bool {
	text := entry.Text
	textNeedle := needle
	if tagOnly {
		textNeedle = tagNeedle
	}

	if !caseSensitive {
		text = strings.ToLower(text)
		needle = strings.ToLower(needle)
		tagNeedle = strings.ToLower(tagNeedle)
		textNeedle = strings.ToLower(textNeedle)
	}

	if tagOnly {
		if includeText && textNeedle != "" && strings.Contains(text, textNeedle) {
			return true
		}
	} else {
		if strings.Contains(text, needle) {
			return true
		}
	}

	for _, tag := range entry.Tags {
		tagValue := tag
		if !caseSensitive {
			tagValue = strings.ToLower(tagValue)
		}
		if tagOnly {
			if tagValue == tagNeedle {
				return true
			}
			continue
		}
		if strings.Contains(tagValue, needle) || tagValue == tagNeedle {
			return true
		}
	}
	return false
}

func printSearchResultsText(cmd *cobra.Command, term string, start time.Time, results []searchResult) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Results for %q in %s\n", term, start.Format("2006-01"))
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
}

func printSearchResultsJSON(cmd *cobra.Command, results []searchResult) error {
	type dto struct {
		Date  string        `json:"date"`
		Index int           `json:"index"`
		Entry logbook.Entry `json:"entry"`
	}

	list := make([]dto, 0, len(results))
	for _, res := range results {
		list = append(list, dto{
			Date:  res.section.Date.Format("2006-01-02"),
			Index: res.index + 1,
			Entry: res.entry,
		})
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(list)
}
