package cli

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/ui"
)

// NewRootCommand creates the top-level Cobra command to host subcommands and TUI launcher.
func NewRootCommand(ctx context.Context, manager *files.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kerja",
		Short: "Track and review daily work logs from your terminal.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := ui.NewModel(ctx, manager)
			if _, err := tea.NewProgram(m).Run(); err != nil {
				return fmt.Errorf("run TUI: %w", err)
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newTodayCommand(ctx, manager),
		newPrevCommand(ctx, manager),
		newNextCommand(ctx, manager),
		newJumpCommand(ctx, manager),
		newListCommand(ctx, manager),
		newSearchCommand(ctx, manager),
		newLogCommand(ctx, manager),
		newTodoCommand(ctx, manager),
		newToggleCommand(ctx, manager),
		newEditCommand(ctx, manager),
		newDeleteCommand(ctx, manager),
	)

	return cmd
}

// ExecuteCommand is a thin wrapper that executes the Cobra root command.
func ExecuteCommand(ctx context.Context) error {
	manager, err := files.NewManager("")
	if err != nil {
		return err
	}
	cmd := NewRootCommand(ctx, manager)
	return cmd.Execute()
}

// Main is a helper used by cmd/kerja/main.go to keep wiring contained in one package.
func Main(ctx context.Context) {
	if err := ExecuteCommand(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
