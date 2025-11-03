package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func newLogCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var (
		dateFlag string
		timeFlag string
	)

	cmd := &cobra.Command{
		Use:   "log [text ... #tags]",
		Short: "Record a completed entry for today.",
		Long:  "log appends a done entry under the target date. Tags can be provided inline via #tag syntax.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("text is required")
			}

			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			entryTime, err := resolveTime(date, timeFlag)
			if err != nil {
				return err
			}

			text, tags := parseTextAndTags(args)
			if text == "" {
				return fmt.Errorf("text is required")
			}

			entry := logbook.Entry{
				Status: logbook.StatusDone,
				Time:   entryTime,
				Text:   text,
				Tags:   tags,
			}

			writer := logbook.NewWriter(manager)
			if err := writer.Append(ctx, date, entry); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Logged %s\n", formatEntry(entry))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&timeFlag, "time", "", "Timestamp in HH:MM (default: current time)")

	return cmd
}

func newTodoCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var (
		dateFlag string
		timeFlag string
	)

	cmd := &cobra.Command{
		Use:   "todo [text ... #tags]",
		Short: "Capture a todo entry for today.",
		Long:  "todo appends an open item under the target date. Tags can be provided inline via #tag syntax.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("text is required")
			}

			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			entryTime, err := resolveTime(date, timeFlag)
			if err != nil {
				return err
			}

			text, tags := parseTextAndTags(args)
			if text == "" {
				return fmt.Errorf("text is required")
			}

			entry := logbook.Entry{
				Status: logbook.StatusTodo,
				Time:   entryTime,
				Text:   text,
				Tags:   tags,
			}

			writer := logbook.NewWriter(manager)
			if err := writer.Append(ctx, date, entry); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added todo %s\n", formatEntry(entry))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&timeFlag, "time", "", "Timestamp in HH:MM (default: current time)")

	return cmd
}

func newToggleCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "toggle <index>",
		Short: "Flip the status of an entry by index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.Atoi(args[0])
			if err != nil || index <= 0 {
				return fmt.Errorf("index must be a positive integer")
			}

			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			writer := logbook.NewWriter(manager)
			entry, err := writer.Toggle(ctx, date, index)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Toggled entry %d: %s\n", index, formatEntry(entry))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")

	return cmd
}

func newDeleteCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "delete <index>",
		Short: "Remove an entry by index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.Atoi(args[0])
			if err != nil || index <= 0 {
				return fmt.Errorf("index must be a positive integer")
			}

			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			writer := logbook.NewWriter(manager)
			entry, err := writer.Delete(ctx, date, index)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted entry %d: %s\n", index, formatEntry(entry))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")

	return cmd
}

func newEditCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var (
		dateFlag   string
		timeFlag   string
		statusFlag string
	)

	cmd := &cobra.Command{
		Use:   "edit <index> [text ... #tags]",
		Short: "Modify an entry by index.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.Atoi(args[0])
			if err != nil || index <= 0 {
				return fmt.Errorf("index must be a positive integer")
			}
			textArgs := args[1:]

			date, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			reader := logbook.NewReader(manager)
			section, err := reader.Section(ctx, date)
			if err != nil {
				return err
			}

			if index > len(section.Entries) {
				return logbook.ErrInvalidIndex
			}

			current := section.Entries[index-1]
			updated := current

			if len(textArgs) > 0 {
				text, tags := parseTextAndTags(textArgs)
				if text != "" {
					updated.Text = text
				} else {
					updated.Text = ""
				}
				updated.Tags = tags
			}

			if timeFlag != "" {
				entryTime, err := resolveTime(date, timeFlag)
				if err != nil {
					return err
				}
				updated.Time = entryTime
			}

			if statusFlag != "" {
				status, err := parseStatusFlag(statusFlag, updated.Status)
				if err != nil {
					return err
				}
				updated.Status = status
			}

			writer := logbook.NewWriter(manager)
			if err := writer.Edit(ctx, date, index, updated); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated entry %d: %s\n", index, formatEntry(updated))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&timeFlag, "time", "", "Timestamp in HH:MM (default: unchanged)")
	cmd.Flags().StringVar(&statusFlag, "status", "", "todo or done (default: unchanged)")

	return cmd
}
