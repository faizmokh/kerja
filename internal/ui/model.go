package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

// Model owns Bubble Tea state for the main TUI experience.
type Model struct {
	ctx    context.Context
	reader *logbook.Reader
	writer *logbook.Writer

	currentDate time.Time
	section     logbook.DateSection
	selected    int

	mode               mode
	inputBuffer        string
	inputLabel         string
	pendingStatus      logbook.Status
	editingIndex       int
	shouldSelectLast   bool
	pendingSelectIndex int

	loading    bool
	statusLine string
	errorLine  string
}

type mode uint8

const (
	modeNormal mode = iota
	modeAddTodo
	modeAddLog
	modeEdit
	modeEditTime
	modeEditStatus
	modeConfirmDelete
)

type sectionLoadedMsg struct {
	date    time.Time
	section logbook.DateSection
	err     error
}

type toggleResultMsg struct {
	index int
	entry logbook.Entry
	err   error
}

type appendResultMsg struct {
	entry logbook.Entry
	err   error
}

type editResultMsg struct {
	index int
	entry logbook.Entry
	err   error
}

type deleteResultMsg struct {
	index int
	err   error
}

type entryInput struct {
	text   string
	tags   []string
	when   *time.Time
	status *logbook.Status
}

// NewModel seeds a Bubble Tea model with required collaborators.
func NewModel(ctx context.Context, manager *files.Manager) Model {
	reader := logbook.NewReader(manager)
	writer := logbook.NewWriter(manager)
	initialDate := today()

	return Model{
		ctx:         ctx,
		reader:      reader,
		writer:      writer,
		currentDate: initialDate,
		section: logbook.DateSection{
			Date: initialDate,
		},
		mode:               modeNormal,
		pendingStatus:      logbook.StatusTodo,
		editingIndex:       -1,
		pendingSelectIndex: -1,
		loading:            true,
		statusLine:         "Loading today's entries...",
	}
}

// Init loads the initial date section.
func (m Model) Init() tea.Cmd {
	return m.loadSectionCmd(m.currentDate)
}

// Update wires TUI state transitions from user input and async commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case sectionLoadedMsg:
		return m.handleSectionLoaded(msg)
	case toggleResultMsg:
		return m.handleToggleResult(msg)
	case appendResultMsg:
		return m.handleAppendResult(msg)
	case editResultMsg:
		return m.handleEditResult(msg)
	case deleteResultMsg:
		return m.handleDeleteResult(msg)
	default:
		return m, nil
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeNormal {
		return m.handleInputKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "down", "j":
		if len(m.section.Entries) == 0 {
			return m, nil
		}
		if m.selected < len(m.section.Entries)-1 {
			m.selected++
			m.statusLine = fmt.Sprintf("Selected entry %d of %d", m.selected+1, len(m.section.Entries))
			m.errorLine = ""
		}
	case "up", "k":
		if len(m.section.Entries) == 0 {
			return m, nil
		}
		if m.selected > 0 {
			m.selected--
			m.statusLine = fmt.Sprintf("Selected entry %d of %d", m.selected+1, len(m.section.Entries))
			m.errorLine = ""
		}
	case "left", "h", "p":
		return m.gotoDate(m.currentDate.AddDate(0, 0, -1))
	case "right", "l", "n":
		return m.gotoDate(m.currentDate.AddDate(0, 0, 1))
	case "t":
		return m.gotoDate(today())
	case "r":
		return m.reload()
	case "x", " ":
		if len(m.section.Entries) == 0 || m.loading {
			return m, nil
		}
		return m.toggleSelected()
	case "a":
		return m.beginAdd(logbook.StatusTodo)
	case "A":
		return m.beginAdd(logbook.StatusDone)
	case "e":
		return m.beginEdit()
	case "T":
		return m.beginEditTime()
	case "S":
		return m.beginEditStatus()
	case "d":
		return m.beginDelete()
	}

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeAddTodo, modeAddLog, modeEdit, modeEditTime, modeEditStatus:
		switch msg.Type {
		case tea.KeyEnter:
			return m.submitInput()
		case tea.KeyEsc:
			return m.cancelInput("Cancelled.")
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = trimLastRune(m.inputBuffer)
			}
		case tea.KeyCtrlU:
			m.inputBuffer = ""
		case tea.KeySpace:
			m.inputBuffer += " "
		case tea.KeyRunes:
			m.inputBuffer += string(msg.Runes)
		}
		return m, nil
	case modeConfirmDelete:
		switch msg.String() {
		case "y", "Y":
			return m.confirmDelete()
		case "n", "N":
			return m.cancelInput("Delete cancelled.")
		case "esc":
			return m.cancelInput("Delete cancelled.")
		case "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) beginAdd(status logbook.Status) (tea.Model, tea.Cmd) {
	m.mode = modeAddTodo
	if status == logbook.StatusDone {
		m.mode = modeAddLog
	}
	m.pendingStatus = status
	m.inputBuffer = ""
	if status == logbook.StatusDone {
		m.inputLabel = "New done entry (text; add @HH:MM, !todo|!done, #tags as needed; Enter to save, Esc to cancel):"
	} else {
		m.inputLabel = "New todo entry (text; add @HH:MM, !todo|!done, #tags as needed; Enter to save, Esc to cancel):"
	}
	m.statusLine = ""
	m.errorLine = ""
	m.editingIndex = -1
	return m, nil
}

func (m Model) beginEdit() (tea.Model, tea.Cmd) {
	if len(m.section.Entries) == 0 {
		return m, nil
	}

	index := m.selected
	entry := m.section.Entries[index]

	m.mode = modeEdit
	m.editingIndex = index
	m.inputBuffer = entryToInput(entry)
	m.inputLabel = fmt.Sprintf("Edit entry %d (adjust text, @HH:MM, !todo|!done, #tags; Enter to save, Esc to cancel):", index+1)
	m.statusLine = ""
	m.errorLine = ""
	return m, nil
}

func (m Model) beginEditTime() (tea.Model, tea.Cmd) {
	if len(m.section.Entries) == 0 {
		return m, nil
	}

	entry := m.section.Entries[m.selected]
	m.mode = modeEditTime
	m.editingIndex = m.selected
	if entry.Time.IsZero() {
		m.inputBuffer = ""
	} else {
		m.inputBuffer = entry.Time.Format("15:04")
	}
	m.inputLabel = fmt.Sprintf("Set time for entry %d (HH:MM, Enter to save, Esc to cancel):", m.selected+1)
	m.statusLine = ""
	m.errorLine = ""
	return m, nil
}

func (m Model) beginEditStatus() (tea.Model, tea.Cmd) {
	if len(m.section.Entries) == 0 {
		return m, nil
	}

	entry := m.section.Entries[m.selected]
	m.mode = modeEditStatus
	m.editingIndex = m.selected
	if entry.Status == logbook.StatusDone {
		m.inputBuffer = "done"
	} else {
		m.inputBuffer = "todo"
	}
	m.inputLabel = fmt.Sprintf("Set status for entry %d (todo|done, Enter to save, Esc to cancel):", m.selected+1)
	m.statusLine = ""
	m.errorLine = ""
	return m, nil
}

func (m Model) beginDelete() (tea.Model, tea.Cmd) {
	if len(m.section.Entries) == 0 {
		return m, nil
	}

	index := m.selected
	m.mode = modeConfirmDelete
	m.editingIndex = index
	m.statusLine = ""
	m.errorLine = ""
	return m, nil
}

func (m Model) submitInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.inputBuffer)
	if input == "" && m.mode != modeEdit {
		m.errorLine = "Entry cannot be empty."
		return m, nil
	}

	switch m.mode {
	case modeAddTodo, modeAddLog:
		parsed, err := parseInputLine(input, m.currentDate)
		if err != nil {
			m.errorLine = err.Error()
			return m, nil
		}
		if parsed.text == "" && len(parsed.tags) == 0 {
			m.errorLine = "Entry cannot be empty."
			return m, nil
		}
		status := m.pendingStatus
		if parsed.status != nil {
			status = *parsed.status
		}
		now := time.Now().In(m.currentDate.Location())
		when := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
		if parsed.when != nil {
			when = *parsed.when
		}
		entry := logbook.Entry{
			Status: status,
			Time:   when,
			Text:   parsed.text,
			Tags:   parsed.tags,
		}
		cmd := m.appendEntryCmd(m.currentDate, entry)
		m.mode = modeNormal
		m.inputBuffer = ""
		m.inputLabel = ""
		m.statusLine = "Saving entry..."
		m.errorLine = ""
		m.pendingSelectIndex = -1
		m.editingIndex = -1
		return m, cmd
	case modeEdit:
		if m.editingIndex < 0 || m.editingIndex >= len(m.section.Entries) {
			return m.cancelInput("No entry selected.")
		}
		original := m.section.Entries[m.editingIndex]
		base := original.Time
		if base.IsZero() {
			base = m.currentDate
		}
		parsed, err := parseInputLine(input, base)
		if err != nil {
			m.errorLine = err.Error()
			return m, nil
		}
		if parsed.text == "" && len(parsed.tags) == 0 && parsed.when == nil && parsed.status == nil {
			m.errorLine = "Entry cannot be empty."
			return m, nil
		}
		updated := original
		updated.Text = parsed.text
		updated.Tags = parsed.tags
		if parsed.when != nil {
			updated.Time = *parsed.when
		}
		if parsed.status != nil {
			updated.Status = *parsed.status
		}
		cmd := m.editEntryCmd(m.currentDate, m.editingIndex, updated)
		m.mode = modeNormal
		m.inputBuffer = ""
		m.inputLabel = ""
		m.statusLine = "Updating entry..."
		m.errorLine = ""
		m.pendingSelectIndex = m.editingIndex
		m.editingIndex = -1
		return m, cmd
	case modeEditTime:
		if m.editingIndex < 0 || m.editingIndex >= len(m.section.Entries) {
			return m.cancelInput("No entry selected.")
		}
		value := strings.TrimSpace(m.inputBuffer)
		if value == "" {
			m.errorLine = "Time cannot be empty."
			return m, nil
		}
		entry := m.section.Entries[m.editingIndex]
		base := entry.Time
		if base.IsZero() {
			base = m.currentDate
		}
		parsed, err := time.ParseInLocation("15:04", value, base.Location())
		if err != nil {
			m.errorLine = fmt.Sprintf("Invalid time %q (expected HH:MM)", value)
			return m, nil
		}
		when := time.Date(base.Year(), base.Month(), base.Day(), parsed.Hour(), parsed.Minute(), 0, 0, base.Location())
		updated := entry
		updated.Time = when
		cmd := m.editEntryCmd(m.currentDate, m.editingIndex, updated)
		m.mode = modeNormal
		m.inputBuffer = ""
		m.inputLabel = ""
		m.statusLine = "Updated time."
		m.errorLine = ""
		m.pendingSelectIndex = m.editingIndex
		m.editingIndex = -1
		return m, cmd
	case modeEditStatus:
		if m.editingIndex < 0 || m.editingIndex >= len(m.section.Entries) {
			return m.cancelInput("No entry selected.")
		}
		value := strings.TrimSpace(strings.ToLower(m.inputBuffer))
		if value == "" {
			m.errorLine = "Status cannot be empty."
			return m, nil
		}
		var status logbook.Status
		switch value {
		case "todo", "t":
			status = logbook.StatusTodo
		case "done", "d":
			status = logbook.StatusDone
		default:
			m.errorLine = fmt.Sprintf("Invalid status %q (expected todo or done)", value)
			return m, nil
		}
		entry := m.section.Entries[m.editingIndex]
		if entry.Status == status {
			m.mode = modeNormal
			m.inputBuffer = ""
			m.inputLabel = ""
			m.statusLine = "Status unchanged."
			m.errorLine = ""
			m.pendingSelectIndex = m.editingIndex
			m.editingIndex = -1
			return m, nil
		}
		updated := entry
		updated.Status = status
		cmd := m.editEntryCmd(m.currentDate, m.editingIndex, updated)
		m.mode = modeNormal
		m.inputBuffer = ""
		m.inputLabel = ""
		m.statusLine = "Updated status."
		m.errorLine = ""
		m.pendingSelectIndex = m.editingIndex
		m.editingIndex = -1
		return m, cmd
	default:
		return m, nil
	}
}

func (m Model) cancelInput(message string) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	m.inputBuffer = ""
	m.inputLabel = ""
	m.pendingStatus = logbook.StatusTodo
	m.editingIndex = -1
	m.shouldSelectLast = false
	m.pendingSelectIndex = -1
	if message != "" {
		m.statusLine = message
	}
	m.errorLine = ""
	return m, nil
}

func (m Model) confirmDelete() (tea.Model, tea.Cmd) {
	if m.editingIndex < 0 || m.editingIndex >= len(m.section.Entries) {
		return m.cancelInput("No entry selected.")
	}
	index := m.editingIndex
	cmd := m.deleteEntryCmd(m.currentDate, index)
	m.mode = modeNormal
	m.statusLine = "Deleting entry..."
	m.errorLine = ""
	m.inputBuffer = ""
	m.inputLabel = ""
	m.editingIndex = -1
	return m, cmd
}

func trimLastRune(input string) string {
	if input == "" {
		return input
	}
	runes := []rune(input)
	return string(runes[:len(runes)-1])
}

func (m Model) handleSectionLoaded(msg sectionLoadedMsg) (tea.Model, tea.Cmd) {
	// Ignore stale results for dates we no longer display.
	if !sameDay(m.currentDate, msg.date) {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errorLine = fmt.Sprintf("Failed to load %s: %v", msg.date.Format("2006-01-02"), msg.err)
		m.statusLine = ""
		return m, nil
	}

	m.errorLine = ""
	section := msg.section
	if section.Date.IsZero() {
		section.Date = msg.date
	}
	m.section = section
	if len(m.section.Entries) == 0 {
		m.selected = 0
		m.statusLine = fmt.Sprintf("%s has no entries.", msg.date.Format("2006-01-02"))
	} else {
		if m.shouldSelectLast {
			m.selected = len(m.section.Entries) - 1
		} else if m.pendingSelectIndex >= 0 {
			if m.pendingSelectIndex >= len(m.section.Entries) {
				m.selected = len(m.section.Entries) - 1
			} else {
				m.selected = m.pendingSelectIndex
			}
		} else if m.selected >= len(m.section.Entries) {
			m.selected = len(m.section.Entries) - 1
		}
		m.statusLine = fmt.Sprintf("Loaded %d entr%s.", len(m.section.Entries), plural(len(m.section.Entries)))
	}
	m.shouldSelectLast = false
	m.pendingSelectIndex = -1
	return m, nil
}

func (m Model) handleToggleResult(msg toggleResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errorLine = fmt.Sprintf("Toggle failed: %v", msg.err)
		m.statusLine = ""
		return m, nil
	}

	if msg.index >= 0 && msg.index < len(m.section.Entries) {
		m.section.Entries[msg.index] = msg.entry
	}

	m.statusLine = fmt.Sprintf("Toggled entry %d (%s).", msg.index+1, msg.entry.Time.Format("15:04"))
	m.errorLine = ""
	return m, nil
}

func (m Model) handleAppendResult(msg appendResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errorLine = fmt.Sprintf("Add failed: %v", msg.err)
		m.statusLine = ""
		return m, nil
	}

	m.errorLine = ""
	m.statusLine = "Entry added."
	m.loading = true
	m.shouldSelectLast = true
	m.pendingSelectIndex = -1
	return m, m.loadSectionCmd(m.currentDate)
}

func (m Model) handleEditResult(msg editResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errorLine = fmt.Sprintf("Edit failed: %v", msg.err)
		m.statusLine = ""
		return m, nil
	}

	m.errorLine = ""
	m.statusLine = fmt.Sprintf("Updated entry %d.", msg.index+1)
	m.loading = true
	m.pendingSelectIndex = msg.index
	return m, m.loadSectionCmd(m.currentDate)
}

func (m Model) handleDeleteResult(msg deleteResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errorLine = fmt.Sprintf("Delete failed: %v", msg.err)
		m.statusLine = ""
		return m, nil
	}

	m.errorLine = ""
	m.statusLine = fmt.Sprintf("Deleted entry %d.", msg.index+1)
	m.loading = true
	m.pendingSelectIndex = msg.index
	return m, m.loadSectionCmd(m.currentDate)
}

func (m Model) gotoDate(date time.Time) (tea.Model, tea.Cmd) {
	if sameDay(m.currentDate, date) {
		return m.reload()
	}

	m.currentDate = date
	m.section = logbook.DateSection{Date: date}
	m.selected = 0
	m.loading = true
	m.statusLine = fmt.Sprintf("Loading %s...", date.Format("2006-01-02"))
	m.errorLine = ""
	m.mode = modeNormal
	m.inputBuffer = ""
	m.inputLabel = ""
	m.editingIndex = -1
	m.pendingSelectIndex = -1
	m.shouldSelectLast = false
	return m, m.loadSectionCmd(date)
}

func (m Model) reload() (tea.Model, tea.Cmd) {
	m.loading = true
	m.statusLine = fmt.Sprintf("Refreshing %s...", m.currentDate.Format("2006-01-02"))
	m.errorLine = ""
	return m, m.loadSectionCmd(m.currentDate)
}

func (m Model) toggleSelected() (tea.Model, tea.Cmd) {
	index := m.selected
	m.statusLine = fmt.Sprintf("Toggling entry %d...", index+1)
	m.errorLine = ""
	return m, m.toggleEntryCmd(m.currentDate, index)
}

func (m Model) loadSectionCmd(date time.Time) tea.Cmd {
	reader := m.reader
	ctx := m.ctx
	return func() tea.Msg {
		section, err := reader.Section(ctx, date)
		if err != nil {
			if errors.Is(err, logbook.ErrSectionNotFound) {
				return sectionLoadedMsg{
					date:    date,
					section: logbook.DateSection{Date: date},
				}
			}
			return sectionLoadedMsg{
				date: date,
				err:  err,
			}
		}
		return sectionLoadedMsg{
			date:    date,
			section: section,
		}
	}
}

func (m Model) toggleEntryCmd(date time.Time, index int) tea.Cmd {
	writer := m.writer
	ctx := m.ctx
	return func() tea.Msg {
		entry, err := writer.Toggle(ctx, date, index+1)
		if err != nil {
			return toggleResultMsg{index: index, err: err}
		}
		return toggleResultMsg{index: index, entry: entry}
	}
}

func (m Model) appendEntryCmd(date time.Time, entry logbook.Entry) tea.Cmd {
	writer := m.writer
	ctx := m.ctx
	return func() tea.Msg {
		if err := writer.Append(ctx, date, entry); err != nil {
			return appendResultMsg{entry: entry, err: err}
		}
		return appendResultMsg{entry: entry}
	}
}

func (m Model) editEntryCmd(date time.Time, index int, entry logbook.Entry) tea.Cmd {
	writer := m.writer
	ctx := m.ctx
	return func() tea.Msg {
		if err := writer.Edit(ctx, date, index+1, entry); err != nil {
			return editResultMsg{index: index, entry: entry, err: err}
		}
		return editResultMsg{index: index, entry: entry}
	}
}

func (m Model) deleteEntryCmd(date time.Time, index int) tea.Cmd {
	writer := m.writer
	ctx := m.ctx
	return func() tea.Msg {
		if _, err := writer.Delete(ctx, date, index+1); err != nil {
			return deleteResultMsg{index: index, err: err}
		}
		return deleteResultMsg{index: index}
	}
}

// View renders the frame.
func (m Model) View() string {
	var b strings.Builder

	header := m.currentDate.Format("Monday, 02 January 2006")
	b.WriteString(header)
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("-", len(header)))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("Loading...\n")
	} else if len(m.section.Entries) == 0 {
		b.WriteString("(no entries)\n")
	} else {
		for i, entry := range m.section.Entries {
			cursor := " "
			if i == m.selected {
				cursor = ">"
			}
			b.WriteString(cursor)
			b.WriteByte(' ')
			b.WriteString(formatEntry(entry))
			b.WriteByte('\n')
		}
	}

	if m.errorLine != "" {
		b.WriteString("\n! ")
		b.WriteString(m.errorLine)
		b.WriteByte('\n')
	} else if m.statusLine != "" {
		b.WriteString("\n")
		b.WriteString(m.statusLine)
		b.WriteByte('\n')
	}

	switch m.mode {
	case modeAddTodo, modeAddLog, modeEdit:
		b.WriteString("\n")
		b.WriteString(m.inputLabel)
		b.WriteByte('\n')
		b.WriteString("> ")
		b.WriteString(m.inputBuffer)
		b.WriteByte('\n')
	case modeConfirmDelete:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Delete entry %d? (y/n, Esc to cancel)", m.editingIndex+1))
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString("Navigation: <-/h/p prev  ->/l/n next  j/k select  t today  r reload")
	b.WriteByte('\n')
	b.WriteString("Actions: space/x toggle  a add todo  A add done  e edit  T set time  S set status  d delete  q quit")
	b.WriteByte('\n')

	return b.String()
}

func today() time.Time {
	now := time.Now().In(time.Local)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func plural(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}

func formatEntry(entry logbook.Entry) string {
	status := "todo"
	if entry.Status == logbook.StatusDone {
		status = "done"
	}

	var builder strings.Builder
	builder.Grow(32 + len(entry.Text) + len(entry.Tags)*6)

	fmt.Fprintf(&builder, "[%s] [%s]", status, entry.Time.Format("15:04"))

	if entry.Text != "" {
		builder.WriteByte(' ')
		builder.WriteString(entry.Text)
	}

	if len(entry.Tags) > 0 {
		builder.WriteString(" (")
		for i, tag := range entry.Tags {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteByte('#')
			builder.WriteString(tag)
		}
		builder.WriteByte(')')
	}

	return builder.String()
}

func entryToInput(entry logbook.Entry) string {
	parts := make([]string, 0, 4+len(entry.Tags))
	if entry.Status == logbook.StatusDone {
		parts = append(parts, "!done")
	} else {
		parts = append(parts, "!todo")
	}
	if !entry.Time.IsZero() {
		parts = append(parts, "@"+entry.Time.Format("15:04"))
	}
	if strings.TrimSpace(entry.Text) != "" {
		parts = append(parts, strings.Fields(entry.Text)...)
	}
	for _, tag := range entry.Tags {
		parts = append(parts, "#"+tag)
	}
	return strings.Join(parts, " ")
}

func parseInputLine(input string, base time.Time) (entryInput, error) {
	result := entryInput{}
	if strings.TrimSpace(input) == "" {
		return result, nil
	}

	var textParts []string
	var tags []string
	for _, token := range strings.Fields(input) {
		switch {
		case strings.HasPrefix(token, "#") && len(token) > 1:
			tags = append(tags, strings.TrimPrefix(token, "#"))
		case strings.HasPrefix(token, "@") && len(token) > 1:
			parsed, err := time.ParseInLocation("15:04", token[1:], base.Location())
			if err != nil {
				return entryInput{}, fmt.Errorf("invalid time %q (expected HH:MM)", token[1:])
			}
			when := time.Date(base.Year(), base.Month(), base.Day(), parsed.Hour(), parsed.Minute(), 0, 0, base.Location())
			result.when = &when
		case strings.HasPrefix(token, "!") && len(token) > 1:
			statusToken := strings.ToLower(token[1:])
			switch statusToken {
			case "todo":
				status := logbook.StatusTodo
				result.status = &status
			case "done":
				status := logbook.StatusDone
				result.status = &status
			default:
				return entryInput{}, fmt.Errorf("invalid status %q (expected !todo or !done)", token)
			}
		default:
			textParts = append(textParts, token)
		}
	}

	result.text = strings.TrimSpace(strings.Join(textParts, " "))
	result.tags = tags
	return result, nil
}
