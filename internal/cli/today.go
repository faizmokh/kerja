package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

func newTodayCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	var dateFlag string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "Show the log entries for today or a specific date.",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDate, err := resolveDate(dateFlag)
			if err != nil {
				return err
			}

			reader := logbook.NewReader(manager)
			section, err := reader.Section(ctx, targetDate)
			if err != nil {
				if errors.Is(err, logbook.ErrSectionNotFound) {
					printMissingSection(cmd, targetDate)
					return nil
				}
				return err
			}

			return printSection(cmd, section)
		},
	}

	cmd.Flags().StringVar(&dateFlag, "date", "", "Target date in YYYY-MM-DD (default: today)")

	return cmd
}
