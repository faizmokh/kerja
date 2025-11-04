package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gumstyle "github.com/charmbracelet/gum/style"
	"github.com/charmbracelet/lipgloss"

	"github.com/faizmokh/kerja/internal/files"
	"github.com/faizmokh/kerja/internal/logbook"
)

const (
	viewportHorizontalPadding = 4
	viewportChromeHeight      = 9
)

var (
	headerStyle        = gumstyle.Styles{Foreground: "213", Bold: true}.ToLipgloss()
	loadingStyle       = gumstyle.Styles{Foreground: "111"}.ToLipgloss()
	statusInfoStyle    = gumstyle.Styles{Foreground: "244"}.ToLipgloss()
	statusErrorStyle   = gumstyle.Styles{Foreground: "196", Bold: true}.ToLipgloss()
	labelStyle         = gumstyle.Styles{Foreground: "244", Bold: true}.ToLipgloss()
	todoBadgeStyle     = gumstyle.Styles{Foreground: "51", Background: "236", Bold: true}.ToLipgloss()
	doneBadgeStyle     = gumstyle.Styles{Foreground: "120", Background: "236", Bold: true}.ToLipgloss()
	timeStyle          = gumstyle.Styles{Foreground: "111"}.ToLipgloss()
	tagStyle           = gumstyle.Styles{Foreground: "177"}.ToLipgloss()
	placeholderStyle   = gumstyle.Styles{Foreground: "241"}.ToLipgloss()
	cursorActiveStyle  = gumstyle.Styles{Foreground: "51", Bold: true}.ToLipgloss()
	cursorPassiveStyle = gumstyle.Styles{Foreground: "238"}.ToLipgloss()

	viewportFrameStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("60")).
				Padding(0, 1)
	selectedEntryStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("57")).
				Foreground(lipgloss.Color("230")).
				Bold(true)
	entryTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	underlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("60"))
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

	viewport      viewport.Model
	viewportReady bool
	help          help.Model
	keys          keyMap
	textInput     textinput.Model
	spinner       spinner.Model
	width         int
	height        int
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	PrevDay    key.Binding
	NextDay    key.Binding
	Today      key.Binding
	Reload     key.Binding
	Toggle     key.Binding
	AddTodo    key.Binding
	AddDone    key.Binding
	Edit       key.Binding
	EditTime   key.Binding
	EditStatus key.Binding
	Delete     key.Binding
	Quit       key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		PrevDay:    key.NewBinding(key.WithKeys("left", "h", "p"), key.WithHelp("←/h/p", "previous day")),
		NextDay:    key.NewBinding(key.WithKeys("right", "l", "n"), key.WithHelp("→/l/n", "next day")),
		Today:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "jump to today")),
		Reload:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Toggle:     key.NewBinding(key.WithKeys("space", "x"), key.WithHelp("space/x", "toggle status")),
		AddTodo:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add todo")),
		AddDone:    key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "add done")),
		Edit:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit entry")),
		EditTime:   key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "edit time")),
		EditStatus: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "edit status")),
		Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete entry")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.AddTodo, k.Edit, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Toggle},
		{k.AddTodo, k.AddDone, k.Edit, k.EditTime, k.EditStatus},
		{k.PrevDay, k.NextDay, k.Today, k.Reload},
		{k.Delete, k.Quit},
	}
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

	vp := viewport.New(0, 0)
	vp.Style = viewportFrameStyle

	helpModel := help.New()
	helpModel.ShowAll = true

	input := textinput.New()
	input.Prompt = cursorPassiveStyle.Render("› ")
	input.Placeholder = "Describe the entry. Use @HH:MM, !todo|!done, #tags"
	input.CharLimit = 512
	input.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	input.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	spin := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))),
	)

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
		viewport:           vp,
		help:               helpModel,
		keys:               newKeyMap(),
		textInput:          input,
		spinner:            spin,
	}
}

// Init loads the initial date section.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return m.spinner.Tick() },
		m.loadSectionCmd(m.currentDate),
	)
}

// Update wires TUI state transitions from user input and async commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
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

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	width := msg.Width - viewportHorizontalPadding
	if width < 20 {
		width = 20
	}

	height := msg.Height - viewportChromeHeight
	if height < 5 {
		height = 5
	}

	m.viewport.Width = width
	m.viewport.Height = height
	m.help.Width = width
	m.viewportReady = true
	m = m.scrollSelectionIntoView()
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeNormal {
		return m.handleInputKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		m = m.moveSelection(1)
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m = m.moveSelection(-1)
		return m, nil
	case key.Matches(msg, m.keys.PrevDay):
		return m.gotoDate(m.currentDate.AddDate(0, 0, -1))
	case key.Matches(msg, m.keys.NextDay):
		return m.gotoDate(m.currentDate.AddDate(0, 0, 1))
	case key.Matches(msg, m.keys.Today):
		return m.gotoDate(today())
	case key.Matches(msg, m.keys.Reload):
		return m.reload()
	case key.Matches(msg, m.keys.Toggle):
		if len(m.section.Entries) == 0 || m.loading {
			return m, nil
		}
		return m.toggleSelected()
	case key.Matches(msg, m.keys.AddTodo):
		return m.beginAdd(logbook.StatusTodo)
	case key.Matches(msg, m.keys.AddDone):
		return m.beginAdd(logbook.StatusDone)
	case key.Matches(msg, m.keys.Edit):
		return m.beginEdit()
	case key.Matches(msg, m.keys.EditTime):
		return m.beginEditTime()
	case key.Matches(msg, m.keys.EditStatus):
		return m.beginEditStatus()
	case key.Matches(msg, m.keys.Delete):
		return m.beginDelete()
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
}

func (m Model) moveSelection(delta int) Model {
	if len(m.section.Entries) == 0 {
		return m
	}

	next := m.selected + delta
	if next < 0 {
		next = 0
	} else if next >= len(m.section.Entries) {
		next = len(m.section.Entries) - 1
	}

	if next != m.selected {
		m.selected = next
		m.statusLine = fmt.Sprintf("Selected entry %d of %d", m.selected+1, len(m.section.Entries))
		m.errorLine = ""
		m = m.scrollSelectionIntoView()
	}

	return m
}

func (m Model) scrollSelectionIntoView() Model {
	if !m.viewportReady || m.viewport.Height <= 0 || len(m.section.Entries) == 0 {
		return m
	}

	if m.selected < m.viewport.YOffset {
		m.viewport.SetYOffset(m.selected)
	} else if m.selected >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(m.selected - m.viewport.Height + 1)
	}

	return m
}

func (m Model) focusTextInput(value, placeholder string) (Model, tea.Cmd) {
	m.inputBuffer = value
	m.textInput.SetValue(value)
	if placeholder != "" {
		m.textInput.Placeholder = placeholder
	}
	m.textInput.Prompt = cursorActiveStyle.Render("› ")
	m.textInput.CursorEnd()
	cmd := m.textInput.Focus()
	return m, cmd
}

func (m Model) resetTextInput() Model {
	m.textInput.Blur()
	m.textInput.SetValue("")
	m.textInput.CursorStart()
	m.textInput.Placeholder = "Describe the entry. Use @HH:MM, !todo|!done, #tags"
	m.textInput.CharLimit = 512
	m.textInput.Prompt = cursorPassiveStyle.Render("› ")
	return m
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeAddTodo, modeAddLog, modeEdit, modeEditTime, modeEditStatus:
		switch msg.Type {
		case tea.KeyEnter:
			m.inputBuffer = m.textInput.Value()
			return m.submitInput()
		case tea.KeyEsc:
			m.textInput.Blur()
			return m.cancelInput("Cancelled.")
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		m.inputBuffer = m.textInput.Value()
		return m, cmd
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
	if status == logbook.StatusDone {
		m.inputLabel = "New done entry (text; add @HH:MM, !todo|!done, #tags as needed; Enter to save, Esc to cancel):"
	} else {
		m.inputLabel = "New todo entry (text; add @HH:MM, !todo|!done, #tags as needed; Enter to save, Esc to cancel):"
	}
	m.statusLine = ""
	m.errorLine = ""
	m.editingIndex = -1
	m.textInput.CharLimit = 512
	placeholder := "Describe the entry. Use @HH:MM, !todo|!done, #tags"
	return m.focusTextInput("", placeholder)
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
	m.textInput.CharLimit = 512
	return m.focusTextInput(m.inputBuffer, "Edit entry data.")
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
	m.textInput.CharLimit = 5
	return m.focusTextInput(m.inputBuffer, "HH:MM")
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
	m.textInput.CharLimit = 4
	return m.focusTextInput(m.inputBuffer, "todo|done")
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
	m.textInput.Blur()
	m.textInput.Prompt = cursorPassiveStyle.Render("› ")
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
		m = m.resetTextInput()
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
		m = m.resetTextInput()
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
		m = m.resetTextInput()
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
			m = m.resetTextInput()
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
		m = m.resetTextInput()
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
	m = m.resetTextInput()
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
		m.viewport.SetYOffset(0)
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
	m = m.scrollSelectionIntoView()
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
	m = m.resetTextInput()
	m.inputBuffer = ""
	m.inputLabel = ""
	m.editingIndex = -1
	m.pendingSelectIndex = -1
	m.shouldSelectLast = false
	m.viewport.SetYOffset(0)
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
	headerText := m.currentDate.Format("Monday, 02 January 2006")
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(headerText),
		underlineStyle.Render(strings.Repeat("─", lipgloss.Width(headerText))),
	)

	var listView string
	if m.loading {
		loading := strings.TrimSpace(fmt.Sprintf("%s %s", m.spinner.View(), loadingStyle.Render("Loading entries...")))
		listView = viewportFrameStyle.Render(loading)
	} else {
		content := m.renderEntries()
		if strings.TrimSpace(content) == "" {
			content = placeholderStyle.Render("(no entries yet)")
		}
		m.viewport.SetContent(content)
		listView = m.viewport.View()
	}

	var status string
	if m.errorLine != "" {
		status = statusErrorStyle.Render("! " + m.errorLine)
	} else if m.statusLine != "" {
		status = statusInfoStyle.Render(m.statusLine)
	}

	var input string
	switch m.mode {
	case modeAddTodo, modeAddLog, modeEdit, modeEditTime, modeEditStatus:
		label := labelStyle.Render(m.inputLabel)
		input = lipgloss.JoinVertical(lipgloss.Left, label, m.textInput.View())
	case modeConfirmDelete:
		prompt := fmt.Sprintf("Delete entry %d? (y/n, Esc to cancel)", m.editingIndex+1)
		input = labelStyle.Render(prompt)
	}

	sections := []string{header, ""}
	if listView != "" {
		sections = append(sections, listView)
	}
	if status != "" {
		sections = append(sections, status)
	}
	if input != "" {
		sections = append(sections, input)
	}
	sections = append(sections, m.help.View(m.keys))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderEntries() string {
	if len(m.section.Entries) == 0 {
		return ""
	}

	lines := make([]string, len(m.section.Entries))
	for i, entry := range m.section.Entries {
		lines[i] = m.renderEntry(entry, i)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderEntry(entry logbook.Entry, index int) string {
	cursor := cursorPassiveStyle.Render(" ")
	if index == m.selected {
		cursor = cursorActiveStyle.Render("›")
	}

	statusBadge := todoBadgeStyle.Render(" TODO ")
	if entry.Status == logbook.StatusDone {
		statusBadge = doneBadgeStyle.Render(" DONE ")
	}

	timeText := "--:--"
	if !entry.Time.IsZero() {
		timeText = entry.Time.Format("15:04")
	}
	timeSegment := timeStyle.Render(timeText)

	text := strings.TrimSpace(entry.Text)
	if text == "" {
		text = "(no description)"
	}
	textSegment := entryTextStyle.Render(text)

	tagSegments := make([]string, len(entry.Tags))
	for i, tag := range entry.Tags {
		tagSegments[i] = tagStyle.Render("#" + tag)
	}

	contentParts := []string{statusBadge, timeSegment, textSegment}
	if len(tagSegments) > 0 {
		contentParts = append(contentParts, strings.Join(tagSegments, " "))
	}

	content := strings.Join(contentParts, " ")
	if index == m.selected {
		content = selectedEntryStyle.Render(content)
	}

	return fmt.Sprintf("%s %s", cursor, content)
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
