package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// Model owns Bubble Tea state for the main TUI experience.
// Real update/view logic will be implemented incrementally.
type Model struct {
	ctx context.Context
}

// NewModel seeds a Bubble Tea model with required collaborators.
func NewModel(ctx context.Context) Model {
	return Model{ctx: ctx}
}

// Init satisfies tea.Model; future work will kick off file loading.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update currently performs no state transitions. Placeholder until TUI wiring is ready.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the frame. A minimal placeholder keeps the app compilable.
func (m Model) View() string {
	return "kerja TUI is under construction.\n"
}

