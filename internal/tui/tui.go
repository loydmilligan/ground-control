// Package tui implements the Bubble Tea TUI for Ground Control.
package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
)

// View modes
type viewMode int

const (
	viewList viewMode = iota
	viewDetail
	viewStateChange
	viewHelp
	viewDump
	viewCreate
	viewSprint
	viewSprintDetail
	viewOrc
	viewOrcDetail
	viewDelegate
)

// Styles
var (
	// Colors
	primaryColor   = lipgloss.Color("99")  // Purple
	secondaryColor = lipgloss.Color("39")  // Cyan
	mutedColor     = lipgloss.Color("241") // Gray
	successColor   = lipgloss.Color("40")  // Green
	warningColor   = lipgloss.Color("214") // Orange
	dangerColor    = lipgloss.Color("196") // Red

	// Layout styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginLeft(2)

	// Detail view styles
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// State styles
	stateStyles = map[types.TaskState]lipgloss.Style{
		types.TaskStateCreated:   lipgloss.NewStyle().Foreground(mutedColor),
		types.TaskStateAssigned:  lipgloss.NewStyle().Foreground(secondaryColor),
		types.TaskStateBlocked:   lipgloss.NewStyle().Foreground(dangerColor),
		types.TaskStateActive:    lipgloss.NewStyle().Foreground(warningColor),
		types.TaskStateWaiting:   lipgloss.NewStyle().Foreground(warningColor),
		types.TaskStateCompleted: lipgloss.NewStyle().Foreground(successColor),
	}

	// State change menu
	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(1, 2)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	menuSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Bold(true).
				Foreground(primaryColor)
)

// Model is the main TUI model.
type Model struct {
	store        *data.Store
	tasks        []types.Task
	list         list.Model
	width        int
	height       int
	err          error
	mode         viewMode
	selectedTask *types.Task
	stateMenuIdx int
	stateOptions []types.TaskState
	dumpInput    textinput.Model

	// Create mode fields
	createStep      int // 0=title, 1=description, 2=type, 3=importance
	createTitle     textinput.Model
	createDesc      textinput.Model
	createTypeIdx   int
	createImpIdx    int
	createTypes     []types.TaskType
	createImps      []types.Importance
	createSuccess   string

	// Sprint mode fields
	sprints        []types.Sprint
	sprintCursor   int
	selectedSprint *types.Sprint

	// Orchestration mode fields
	sessions        []types.Session
	sessionCursor   int
	selectedSession *types.Session

	// Delegation mode fields
	delegateInteractions int
	delegateConfirm      bool
	delegateStatus       string

	// Confirmation mode
	confirmRun     bool
	confirmMessage string
}

// taskItem wraps a task for the list component.
type taskItem struct {
	task types.Task
}

func (i taskItem) Title() string {
	style := stateStyles[i.task.State]
	stateIcon := stateIcon(i.task.State)
	return fmt.Sprintf("%s %s", style.Render(stateIcon), i.task.Title)
}

func (i taskItem) Description() string {
	// Complexity dots
	dots := ""
	for j := 0; j < i.task.Complexity; j++ {
		dots += "●"
	}
	for j := i.task.Complexity; j < 5; j++ {
		dots += "○"
	}
	return fmt.Sprintf("%s  %s  %s", i.task.Type, dots, i.task.Importance)
}

func (i taskItem) FilterValue() string { return i.task.Title }

func stateIcon(state types.TaskState) string {
	switch state {
	case types.TaskStateCompleted:
		return "✓"
	case types.TaskStateActive:
		return "▶"
	case types.TaskStateBlocked:
		return "⊘"
	case types.TaskStateWaiting:
		return "◷"
	default:
		return "○"
	}
}

// New creates a new TUI model.
func New(store *data.Store) Model {
	tasks, err := store.LoadTasks()
	if err != nil {
		return Model{store: store, err: err}
	}

	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Ground Control"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	ti := textinput.New()
	ti.Placeholder = "Enter your idea..."
	ti.Width = 60

	// Create mode text inputs
	titleInput := textinput.New()
	titleInput.Placeholder = "Task title..."
	titleInput.Width = 60

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)..."
	descInput.Width = 60

	return Model{
		store:        store,
		tasks:        tasks,
		list:         l,
		mode:         viewList,
		stateOptions: []types.TaskState{
			types.TaskStateCreated,
			types.TaskStateAssigned,
			types.TaskStateActive,
			types.TaskStateBlocked,
			types.TaskStateWaiting,
			types.TaskStateCompleted,
		},
		dumpInput:   ti,
		createTitle: titleInput,
		createDesc:  descInput,
		createTypes: []types.TaskType{
			types.TaskTypeSimple,
			types.TaskTypeCoding,
			types.TaskTypeResearch,
			types.TaskTypeAIPlanning,
			types.TaskTypeHumanInput,
		},
		createImps: []types.Importance{
			types.ImportanceHigh,
			types.ImportanceMedium,
			types.ImportanceLow,
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)

	case orchestrationStartedMsg:
		if msg.success {
			// Orchestration started successfully
			m.createSuccess = fmt.Sprintf("Orchestration started for task %s", msg.taskID)
			// Switch to orchestration view
			sessions, err := m.store.LoadSessions()
			if err != nil {
				m.err = err
			} else {
				m.sessions = sessions
				m.sessionCursor = 0
				m.mode = viewOrc
			}
			m.selectedTask = nil
		} else {
			// Failed to start orchestration
			m.err = fmt.Errorf("failed to start orchestration: %w", msg.err)
		}
		return m, nil

	case delegateResultMsg:
		if msg.success {
			m.delegateStatus = "Delegation started! Monitor pane added to your window."
			m.createSuccess = "Supervised delegation started"
			m.mode = viewList
		} else {
			m.err = fmt.Errorf("failed to start delegation: %w", msg.err)
			m.mode = viewList
		}
		return m, nil
	}

	if m.mode == viewList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.mode == viewDump {
		var cmd tea.Cmd
		m.dumpInput, cmd = m.dumpInput.Update(msg)
		return m, cmd
	}

	if m.mode == viewCreate {
		var cmd tea.Cmd
		if m.createStep == 0 {
			m.createTitle, cmd = m.createTitle.Update(msg)
		} else if m.createStep == 1 {
			m.createDesc, cmd = m.createDesc.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "?":
		if m.mode == viewHelp {
			m.mode = viewList
		} else {
			m.mode = viewHelp
		}
		return m, nil
	}

	// Mode-specific keys
	switch m.mode {
	case viewList:
		return m.handleListKeys(key, msg)
	case viewDetail:
		return m.handleDetailKeys(key)
	case viewStateChange:
		return m.handleStateChangeKeys(key)
	case viewHelp:
		return m.handleHelpKeys(key)
	case viewDump:
		return m.handleDumpKeys(key)
	case viewCreate:
		return m.handleCreateKeys(key)
	case viewSprint:
		return m.handleSprintKeys(key)
	case viewSprintDetail:
		return m.handleSprintDetailKeys(key)
	case viewOrc:
		return m.handleOrcKeys(key)
	case viewOrcDetail:
		return m.handleOrcDetailKeys(key)
	case viewDelegate:
		return m.handleDelegateKeys(key)
	}

	return m, nil
}

func (m Model) handleListKeys(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirmation flow
	if m.confirmRun {
		switch key {
		case "y":
			// Confirmed - start orchestration
			if m.selectedTask != nil {
				m.confirmRun = false
				return m, m.startOrchestration(m.selectedTask.ID)
			}
			m.confirmRun = false
			return m, nil
		case "n", "esc":
			// Cancelled
			m.confirmRun = false
			m.selectedTask = nil
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "q":
		return m, tea.Quit
	case "tab":
		// Switch to sprint view
		sprints, err := m.store.LoadSprints()
		if err != nil {
			m.err = err
		} else {
			m.sprints = sprints
			m.sprintCursor = 0
			m.mode = viewSprint
		}
		return m, nil
	case "o":
		// Switch to orchestration view
		sessions, err := m.store.LoadSessions()
		if err != nil {
			m.err = err
		} else {
			m.sessions = sessions
			m.sessionCursor = 0
			m.mode = viewOrc
		}
		return m, nil
	case "r":
		// Run orchestration
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.selectedTask = &item.task
			m.confirmRun = true
			m.confirmMessage = fmt.Sprintf("Run orchestration on: %s? (y/n)", item.task.Title)
		}
		return m, nil
	case "enter":
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.selectedTask = &item.task
			m.mode = viewDetail
		}
		return m, nil
	case "s":
		if item, ok := m.list.SelectedItem().(taskItem); ok {
			m.selectedTask = &item.task
			m.mode = viewStateChange
			// Set menu index to current state
			for i, s := range m.stateOptions {
				if s == item.task.State {
					m.stateMenuIdx = i
					break
				}
			}
		}
		return m, nil
	case "d":
		m.mode = viewDump
		m.dumpInput.Focus()
		return m, textinput.Blink
	case "D":
		// Start delegation mode
		m.mode = viewDelegate
		m.delegateInteractions = 5 // Default
		m.delegateConfirm = false
		m.delegateStatus = ""
		return m, nil
	case "c":
		m.mode = viewCreate
		m.createStep = 0
		m.createTitle.Reset()
		m.createDesc.Reset()
		m.createTypeIdx = 0
		m.createImpIdx = 1 // Default to medium
		m.createSuccess = ""
		m.createTitle.Focus()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "b":
		m.mode = viewList
		m.selectedTask = nil
	case "s":
		m.mode = viewStateChange
		for i, s := range m.stateOptions {
			if s == m.selectedTask.State {
				m.stateMenuIdx = i
				break
			}
		}
	}
	return m, nil
}

func (m Model) handleStateChangeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "b":
		m.mode = viewDetail
	case "up", "k":
		if m.stateMenuIdx > 0 {
			m.stateMenuIdx--
		}
	case "down", "j":
		if m.stateMenuIdx < len(m.stateOptions)-1 {
			m.stateMenuIdx++
		}
	case "enter":
		// Update task state
		newState := m.stateOptions[m.stateMenuIdx]
		if err := m.updateTaskState(m.selectedTask.ID, newState); err != nil {
			m.err = err
		} else {
			m.selectedTask.State = newState
			m.refreshTasks()
		}
		m.mode = viewDetail
	}
	return m, nil
}

func (m Model) handleHelpKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "b", "?":
		m.mode = viewList
	}
	return m, nil
}

func (m Model) handleDumpKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		content := m.dumpInput.Value()
		if content != "" {
			if _, err := m.store.AddBrainDump(content); err != nil {
				m.err = err
			}
			m.dumpInput.Reset()
		}
		m.dumpInput.Blur()
		m.mode = viewList
		return m, nil
	case "esc":
		m.dumpInput.Reset()
		m.dumpInput.Blur()
		m.mode = viewList
		return m, nil
	}
	return m, nil
}

func (m Model) handleCreateKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		// Cancel creation
		m.createTitle.Reset()
		m.createDesc.Reset()
		m.createTitle.Blur()
		m.createDesc.Blur()
		m.mode = viewList
		return m, nil

	case "enter":
		if m.createStep == 0 {
			// Title step - move to description
			if m.createTitle.Value() == "" {
				return m, nil // Title is required
			}
			m.createTitle.Blur()
			m.createDesc.Focus()
			m.createStep = 1
			return m, textinput.Blink
		} else if m.createStep == 1 {
			// Description step - move to type selection
			m.createDesc.Blur()
			m.createStep = 2
			return m, nil
		} else if m.createStep == 2 {
			// Type selection - move to importance
			m.createStep = 3
			return m, nil
		} else if m.createStep == 3 {
			// Importance selection - create task
			if err := m.createTask(); err != nil {
				m.err = err
			} else {
				m.createSuccess = "Task created successfully!"
				m.refreshTasks()
			}
			m.createTitle.Reset()
			m.createDesc.Reset()
			m.createTitle.Blur()
			m.createDesc.Blur()
			m.mode = viewList
			return m, nil
		}

	case "up", "k":
		if m.createStep == 2 {
			// Navigate type options
			if m.createTypeIdx > 0 {
				m.createTypeIdx--
			}
		} else if m.createStep == 3 {
			// Navigate importance options
			if m.createImpIdx > 0 {
				m.createImpIdx--
			}
		}
		return m, nil

	case "down", "j":
		if m.createStep == 2 {
			// Navigate type options
			if m.createTypeIdx < len(m.createTypes)-1 {
				m.createTypeIdx++
			}
		} else if m.createStep == 3 {
			// Navigate importance options
			if m.createImpIdx < len(m.createImps)-1 {
				m.createImpIdx++
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleSprintKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "tab":
		m.mode = viewList
		m.selectedSprint = nil
	case "up", "k":
		if m.sprintCursor > 0 {
			m.sprintCursor--
		}
	case "down", "j":
		if m.sprintCursor < len(m.sprints)-1 {
			m.sprintCursor++
		}
	case "enter":
		if m.sprintCursor < len(m.sprints) {
			m.selectedSprint = &m.sprints[m.sprintCursor]
			m.mode = viewSprintDetail
		}
	}
	return m, nil
}

func (m Model) handleSprintDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "b":
		m.mode = viewSprint
		m.selectedSprint = nil
	case "tab":
		m.mode = viewList
		m.selectedSprint = nil
	}
	return m, nil
}

func (m Model) handleOrcKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc":
		m.mode = viewList
		m.selectedSession = nil
	case "up", "k":
		if m.sessionCursor > 0 {
			m.sessionCursor--
		}
	case "down", "j":
		if m.sessionCursor < len(m.sessions)-1 {
			m.sessionCursor++
		}
	case "enter":
		if m.sessionCursor < len(m.sessions) {
			m.selectedSession = &m.sessions[m.sessionCursor]
			m.mode = viewOrcDetail
		}
	case "r":
		// Refresh sessions
		sessions, err := m.store.LoadSessions()
		if err != nil {
			m.err = err
		} else {
			m.sessions = sessions
		}
	}
	return m, nil
}

func (m Model) handleOrcDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "b":
		m.mode = viewOrc
		m.selectedSession = nil
	case "r":
		// Refresh sessions
		sessions, err := m.store.LoadSessions()
		if err != nil {
			m.err = err
		} else {
			m.sessions = sessions
			// Update selected session if it still exists
			if m.selectedSession != nil {
				for i := range m.sessions {
					if m.sessions[i].ID == m.selectedSession.ID {
						m.selectedSession = &m.sessions[i]
						break
					}
				}
			}
		}
	}
	return m, nil
}

func (m Model) handleDelegateKeys(key string) (tea.Model, tea.Cmd) {
	if m.delegateConfirm {
		switch key {
		case "y":
			// Start supervised delegation
			m.delegateConfirm = false
			m.delegateStatus = "Starting delegation..."
			return m, m.startDelegation()
		case "n", "q", "esc":
			m.delegateConfirm = false
			m.mode = viewList
		}
		return m, nil
	}

	switch key {
	case "q", "esc", "b":
		m.mode = viewList
	case "up", "k":
		if m.delegateInteractions < 20 {
			m.delegateInteractions++
		}
	case "down", "j":
		if m.delegateInteractions > 1 {
			m.delegateInteractions--
		}
	case "enter":
		m.delegateConfirm = true
	}
	return m, nil
}

func (m *Model) startDelegation() tea.Cmd {
	return func() tea.Msg {
		// Run the delegation script
		projectRoot := m.store.GetDataDir()
		projectRoot = projectRoot[:len(projectRoot)-5] // Remove "/data"
		script := projectRoot + "/scripts/start-delegation-session.sh"

		cmd := exec.Command(script, fmt.Sprintf("%d", m.delegateInteractions))
		cmd.Dir = projectRoot

		if err := cmd.Run(); err != nil {
			return delegateResultMsg{err: err}
		}
		return delegateResultMsg{success: true}
	}
}

type delegateResultMsg struct {
	success bool
	err     error
}

func (m *Model) updateTaskState(taskID string, newState types.TaskState) error {
	tasks, err := m.store.LoadTasks()
	if err != nil {
		return err
	}

	for i, t := range tasks {
		if t.ID == taskID {
			tasks[i].State = newState
			break
		}
	}

	return m.store.SaveTasks(tasks)
}

func (m *Model) refreshTasks() {
	tasks, err := m.store.LoadTasks()
	if err != nil {
		m.err = err
		return
	}
	m.tasks = tasks

	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	m.list.SetItems(items)
}

func (m *Model) createTask() error {
	tasks, err := m.store.LoadTasks()
	if err != nil {
		return err
	}

	now := time.Now()
	task := types.Task{
		ID:              fmt.Sprintf("task_%d", now.UnixMilli()),
		Title:           m.createTitle.Value(),
		Description:     m.createDesc.Value(),
		Type:            m.createTypes[m.createTypeIdx],
		Agent:           nil,
		AssignedHuman:   "matt",
		AutonomyLevel:   types.AutonomyCheckpoints,
		Complexity:      1,
		Importance:      m.createImps[m.createImpIdx],
		DueDate:         nil,
		DueUrgency:      types.DueUrgencyNone,
		Context: types.TaskContext{
			Background:   "",
			Requirements: []string{},
			Constraints:  []string{},
			RelatedTasks: []string{},
			ProjectID:    nil,
		},
		Topics:          []string{},
		State:           types.TaskStateCreated,
		BlockedBy:       []string{},
		ConversationID:  nil,
		Outputs:         []types.TaskOutput{},
		SuggestedNext:   []string{},
		AfterCompletion: types.AfterCompletionTaskmasterReview,
		Verification: types.Verification{
			Type: types.VerificationNone,
		},
		ProjectID:     nil,
		Tags:          []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
		CompletedAt:   nil,
		ActualMinutes: nil,
		TokensUsed:    nil,
		LinesChanged:  nil,
	}

	tasks = append(tasks, task)
	return m.store.SaveTasks(tasks)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	switch m.mode {
	case viewDetail:
		return m.viewDetail()
	case viewStateChange:
		return m.viewStateChange()
	case viewHelp:
		return m.viewHelp()
	case viewDump:
		return m.viewDump()
	case viewCreate:
		return m.viewCreate()
	case viewSprint:
		return m.viewSprintList()
	case viewSprintDetail:
		return m.viewSprintDetail()
	case viewOrc:
		return m.viewOrc()
	case viewOrcDetail:
		return m.viewOrcDetail()
	case viewDelegate:
		return m.viewDelegate()
	default:
		return m.viewList()
	}
}

func (m Model) viewList() string {
	// Show confirmation prompt if confirmRun is true
	if m.confirmRun {
		confirmStyle := lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)
		view := m.list.View() + "\n" + confirmStyle.Render(m.confirmMessage)
		return view
	}

	help := helpStyle.Render("q: quit • /: filter • enter: details • s: state • r: run • c: create • d: dump • D: delegate • o: orc • tab: sprints • ?: help")
	view := m.list.View() + "\n" + help

	// Show success message if present
	if m.createSuccess != "" {
		success := lipgloss.NewStyle().Foreground(successColor).Render(m.createSuccess)
		view = m.list.View() + "\n" + success + "\n" + help
		// Clear the success message after showing
		m.createSuccess = ""
	}

	return view
}

func (m Model) viewDetail() string {
	if m.selectedTask == nil {
		return "No task selected"
	}
	t := m.selectedTask

	var b strings.Builder

	// Title
	b.WriteString(detailTitleStyle.Render(t.Title))
	b.WriteString("\n\n")

	// State and metadata
	stateStyle := stateStyles[t.State]
	b.WriteString(detailLabelStyle.Render("State: "))
	b.WriteString(stateStyle.Render(string(t.State)))
	b.WriteString("   ")
	b.WriteString(detailLabelStyle.Render("Type: "))
	b.WriteString(detailValueStyle.Render(string(t.Type)))
	b.WriteString("   ")
	b.WriteString(detailLabelStyle.Render("Complexity: "))
	dots := ""
	for j := 0; j < t.Complexity; j++ {
		dots += "●"
	}
	for j := t.Complexity; j < 5; j++ {
		dots += "○"
	}
	b.WriteString(detailValueStyle.Render(dots))
	b.WriteString("\n\n")

	// Description
	b.WriteString(detailLabelStyle.Render("Description:"))
	b.WriteString("\n")
	b.WriteString(detailValueStyle.Render(t.Description))
	b.WriteString("\n\n")

	// Context
	if t.Context.Background != "" {
		b.WriteString(detailLabelStyle.Render("Background:"))
		b.WriteString("\n")
		b.WriteString(detailValueStyle.Render(t.Context.Background))
		b.WriteString("\n\n")
	}

	// Requirements
	if len(t.Context.Requirements) > 0 {
		b.WriteString(detailLabelStyle.Render("Requirements:"))
		b.WriteString("\n")
		for _, req := range t.Context.Requirements {
			b.WriteString(detailValueStyle.Render("  • " + req))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Outputs
	if len(t.Outputs) > 0 {
		b.WriteString(detailLabelStyle.Render("Outputs:"))
		b.WriteString("\n")
		for _, out := range t.Outputs {
			icon := "○"
			if out.Exists {
				icon = "✓"
			}
			path := out.Path
			if path == "" {
				path = "(no path)"
			}
			b.WriteString(detailValueStyle.Render(fmt.Sprintf("  %s %s — %s", icon, path, out.Description)))
			b.WriteString("\n")
		}
	}

	// Wrap in box
	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("b/esc: back • s: change state • q: quit")

	return content + "\n\n" + help
}

func (m Model) viewStateChange() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	var b strings.Builder
	b.WriteString(detailLabelStyle.Render("Change state for: "))
	b.WriteString(m.selectedTask.Title)
	b.WriteString("\n\n")

	for i, state := range m.stateOptions {
		style := menuItemStyle
		prefix := "  "
		if i == m.stateMenuIdx {
			style = menuSelectedStyle
			prefix = "▸ "
		}

		icon := stateIcon(state)
		stateStyle := stateStyles[state]
		line := fmt.Sprintf("%s%s %s", prefix, stateStyle.Render(icon), state)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	content := menuStyle.Render(b.String())
	help := helpStyle.Render("↑/k: up • ↓/j: down • enter: select • b/esc: cancel")

	return content + "\n\n" + help
}

func (m Model) viewHelp() string {
	help := `
  Ground Control TUI — Keyboard Shortcuts

  LIST VIEW
    enter     View task details
    s         Change task state
    r         Run (orchestrate) selected task
    c         Create new task
    d         Brain dump
    o         Orchestration sessions
    tab       Switch to sprint view
    /         Filter tasks
    q         Quit

  DETAIL VIEW
    s         Change task state
    b, esc    Back to list
    q         Quit

  SPRINT VIEW
    ↑, k      Move up
    ↓, j      Move down
    enter     View sprint tasks
    tab, esc  Back to task list
    q         Quit

  SPRINT DETAIL VIEW
    b, esc    Back to sprint list
    tab       Back to task list
    q         Quit

  ORCHESTRATION VIEW
    ↑, k      Move up
    ↓, j      Move down
    enter     View session details
    r         Refresh sessions
    esc, q    Back to task list

  ORCHESTRATION DETAIL VIEW
    r         Refresh session
    b, esc    Back to session list
    q         Quit

  STATE CHANGE
    ↑, k      Move up
    ↓, j      Move down
    enter     Confirm selection
    b, esc    Cancel

  CREATE TASK
    enter     Next step / Create
    ↑, k      Move up (dropdowns)
    ↓, j      Move down (dropdowns)
    esc       Cancel

  BRAIN DUMP
    enter     Save idea
    esc       Cancel

  GLOBAL
    ?         Toggle this help
    ctrl+c    Quit
`
	content := detailBoxStyle.Width(60).Render(help)
	footer := helpStyle.Render("Press any key to close")
	return content + "\n\n" + footer
}

func (m Model) viewDump() string {
	var b strings.Builder

	title := detailTitleStyle.Render("Brain Dump")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(detailLabelStyle.Render("Capture your idea:"))
	b.WriteString("\n\n")
	b.WriteString(m.dumpInput.View())
	b.WriteString("\n")

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("enter: save • esc: cancel")

	return content + "\n\n" + help
}

func (m Model) viewCreate() string {
	var b strings.Builder

	title := detailTitleStyle.Render("Create Task")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Step 0: Title
	stepStyle := detailLabelStyle
	if m.createStep == 0 {
		stepStyle = stepStyle.Foreground(primaryColor)
	} else {
		stepStyle = stepStyle.Foreground(mutedColor)
	}
	b.WriteString(stepStyle.Render("1. Title (required):"))
	b.WriteString("\n")
	if m.createStep == 0 {
		b.WriteString(m.createTitle.View())
	} else {
		value := m.createTitle.Value()
		if value == "" {
			value = "(empty)"
		}
		b.WriteString(detailValueStyle.Render(value))
	}
	b.WriteString("\n\n")

	// Step 1: Description
	stepStyle = detailLabelStyle
	if m.createStep == 1 {
		stepStyle = stepStyle.Foreground(primaryColor)
	} else {
		stepStyle = stepStyle.Foreground(mutedColor)
	}
	b.WriteString(stepStyle.Render("2. Description (optional):"))
	b.WriteString("\n")
	if m.createStep == 1 {
		b.WriteString(m.createDesc.View())
	} else {
		value := m.createDesc.Value()
		if value == "" {
			value = "(none)"
		}
		b.WriteString(detailValueStyle.Render(value))
	}
	b.WriteString("\n\n")

	// Step 2: Type
	stepStyle = detailLabelStyle
	if m.createStep == 2 {
		stepStyle = stepStyle.Foreground(primaryColor)
	} else {
		stepStyle = stepStyle.Foreground(mutedColor)
	}
	b.WriteString(stepStyle.Render("3. Type:"))
	b.WriteString("\n")
	if m.createStep == 2 {
		for i, t := range m.createTypes {
			style := menuItemStyle
			prefix := "  "
			if i == m.createTypeIdx {
				style = menuSelectedStyle
				prefix = "▸ "
			}
			b.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, t)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(detailValueStyle.Render(string(m.createTypes[m.createTypeIdx])))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Step 3: Importance
	stepStyle = detailLabelStyle
	if m.createStep == 3 {
		stepStyle = stepStyle.Foreground(primaryColor)
	} else {
		stepStyle = stepStyle.Foreground(mutedColor)
	}
	b.WriteString(stepStyle.Render("4. Importance:"))
	b.WriteString("\n")
	if m.createStep == 3 {
		for i, imp := range m.createImps {
			style := menuItemStyle
			prefix := "  "
			if i == m.createImpIdx {
				style = menuSelectedStyle
				prefix = "▸ "
			}
			b.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, imp)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(detailValueStyle.Render(string(m.createImps[m.createImpIdx])))
		b.WriteString("\n")
	}

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())

	var help string
	if m.createStep == 0 || m.createStep == 1 {
		help = helpStyle.Render("enter: next • esc: cancel")
	} else {
		help = helpStyle.Render("↑/k: up • ↓/j: down • enter: next/create • esc: cancel")
	}

	return content + "\n\n" + help
}

func (m Model) viewSprintList() string {
	var b strings.Builder

	title := titleStyle.Render("Sprints")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.sprints) == 0 {
		noSprintsStyle := lipgloss.NewStyle().Foreground(mutedColor)
		b.WriteString(noSprintsStyle.Render("  No sprints found"))
		b.WriteString("\n")
	} else {
		for i, sprint := range m.sprints {
			style := menuItemStyle
			prefix := "  "
			if i == m.sprintCursor {
				style = menuSelectedStyle
				prefix = "▸ "
			}

			// Status icon
			var statusIcon string
			var statusColor lipgloss.Color
			switch sprint.Status {
			case types.SprintStatusActive:
				statusIcon = "▶"
				statusColor = warningColor
			case types.SprintStatusPaused:
				statusIcon = "⏸"
				statusColor = mutedColor
			case types.SprintStatusCompleted:
				statusIcon = "✓"
				statusColor = successColor
			}

			// Task count
			taskCount := fmt.Sprintf("(%d tasks)", len(sprint.TaskIDs))

			// Build the line
			iconPart := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
			namePart := sprint.Name
			countPart := lipgloss.NewStyle().Foreground(mutedColor).Render(taskCount)

			line := fmt.Sprintf("%s%s %s %s", prefix, iconPart, namePart, countPart)

			// Add goal if present
			if sprint.Goal != "" {
				goalPart := lipgloss.NewStyle().Foreground(secondaryColor).Render(" → " + sprint.Goal)
				line += goalPart
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("↑/k: up • ↓/j: down • enter: view tasks • tab/esc: back to tasks • q: quit")

	return content + "\n\n" + help
}

func (m Model) viewSprintDetail() string {
	if m.selectedSprint == nil {
		return "No sprint selected"
	}

	var b strings.Builder

	// Sprint header
	b.WriteString(detailTitleStyle.Render(m.selectedSprint.Name))
	b.WriteString("\n\n")

	// Status
	var statusIcon string
	var statusColor lipgloss.Color
	switch m.selectedSprint.Status {
	case types.SprintStatusActive:
		statusIcon = "▶"
		statusColor = warningColor
	case types.SprintStatusPaused:
		statusIcon = "⏸"
		statusColor = mutedColor
	case types.SprintStatusCompleted:
		statusIcon = "✓"
		statusColor = successColor
	}

	b.WriteString(detailLabelStyle.Render("Status: "))
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	b.WriteString(statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, m.selectedSprint.Status)))
	b.WriteString("\n\n")

	// Goal
	if m.selectedSprint.Goal != "" {
		b.WriteString(detailLabelStyle.Render("Goal:"))
		b.WriteString("\n")
		b.WriteString(detailValueStyle.Render(m.selectedSprint.Goal))
		b.WriteString("\n\n")
	}

	// Description
	if m.selectedSprint.Description != "" {
		b.WriteString(detailLabelStyle.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(detailValueStyle.Render(m.selectedSprint.Description))
		b.WriteString("\n\n")
	}

	// Tasks
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Tasks (%d):", len(m.selectedSprint.TaskIDs))))
	b.WriteString("\n")

	if len(m.selectedSprint.TaskIDs) == 0 {
		b.WriteString(detailValueStyle.Render("  No tasks in this sprint"))
		b.WriteString("\n")
	} else {
		// Build a map of task IDs for quick lookup
		taskMap := make(map[string]types.Task)
		for _, task := range m.tasks {
			taskMap[task.ID] = task
		}

		// Display tasks
		for _, taskID := range m.selectedSprint.TaskIDs {
			task, exists := taskMap[taskID]
			if !exists {
				b.WriteString(detailValueStyle.Render(fmt.Sprintf("  • %s (not found)", taskID)))
				b.WriteString("\n")
				continue
			}

			// Task with status icon
			stateStyle := stateStyles[task.State]
			icon := stateIcon(task.State)
			taskLine := fmt.Sprintf("  %s %s", stateStyle.Render(icon), task.Title)
			b.WriteString(detailValueStyle.Render(taskLine))
			b.WriteString("\n")
		}
	}

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("b/esc: back to sprints • tab: back to tasks • q: quit")

	return content + "\n\n" + help
}

func (m Model) viewOrc() string {
	var b strings.Builder

	title := titleStyle.Render("Orchestration Sessions")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		noSessionsStyle := lipgloss.NewStyle().Foreground(mutedColor)
		b.WriteString(noSessionsStyle.Render("  No sessions found"))
		b.WriteString("\n")
	} else {
		// Sort sessions: running first, then by start time
		sortedSessions := make([]types.Session, len(m.sessions))
		copy(sortedSessions, m.sessions)

		// Simple bubble sort: running first
		for i := 0; i < len(sortedSessions)-1; i++ {
			for j := 0; j < len(sortedSessions)-i-1; j++ {
				if sortedSessions[j].Status != types.SessionStatusRunning &&
					sortedSessions[j+1].Status == types.SessionStatusRunning {
					sortedSessions[j], sortedSessions[j+1] = sortedSessions[j+1], sortedSessions[j]
				}
			}
		}

		for i, session := range sortedSessions {
			style := menuItemStyle
			prefix := "  "
			if i == m.sessionCursor {
				style = menuSelectedStyle
				prefix = "▸ "
			}

			// Status icon
			var statusIcon string
			var statusColor lipgloss.Color
			switch session.Status {
			case types.SessionStatusRunning:
				statusIcon = "⚙"
				statusColor = warningColor
			case types.SessionStatusCompleted:
				statusIcon = "✓"
				statusColor = successColor
			case types.SessionStatusFailed:
				statusIcon = "✗"
				statusColor = dangerColor
			case types.SessionStatusCancelled:
				statusIcon = "⊘"
				statusColor = mutedColor
			}

			// Session info
			iconPart := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
			idPart := session.ID
			taskCount := fmt.Sprintf("(%d tasks)", len(session.TaskIDs))
			countPart := lipgloss.NewStyle().Foreground(mutedColor).Render(taskCount)

			line := fmt.Sprintf("%s%s %s %s", prefix, iconPart, idPart, countPart)

			// Add current task and stage if running
			if session.Status == types.SessionStatusRunning && session.CurrentTaskID != nil {
				currentPart := lipgloss.NewStyle().Foreground(secondaryColor).Render(
					fmt.Sprintf(" → %s", *session.CurrentTaskID))
				line += currentPart

				if session.CurrentStage != nil {
					stagePart := lipgloss.NewStyle().Foreground(mutedColor).Render(
						fmt.Sprintf(" [%s]", *session.CurrentStage))
					line += stagePart
				}
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("↑/k: up • ↓/j: down • enter: details • r: refresh • esc/q: back")

	return content + "\n\n" + help
}

func (m Model) viewOrcDetail() string {
	if m.selectedSession == nil {
		return "No session selected"
	}

	var b strings.Builder

	// Session header
	b.WriteString(detailTitleStyle.Render("Session: " + m.selectedSession.ID))
	b.WriteString("\n\n")

	// Status
	var statusIcon string
	var statusColor lipgloss.Color
	switch m.selectedSession.Status {
	case types.SessionStatusRunning:
		statusIcon = "⚙"
		statusColor = warningColor
	case types.SessionStatusCompleted:
		statusIcon = "✓"
		statusColor = successColor
	case types.SessionStatusFailed:
		statusIcon = "✗"
		statusColor = dangerColor
	case types.SessionStatusCancelled:
		statusIcon = "⊘"
		statusColor = mutedColor
	}

	b.WriteString(detailLabelStyle.Render("Status: "))
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	b.WriteString(statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, m.selectedSession.Status)))
	b.WriteString("\n\n")

	// Timing
	b.WriteString(detailLabelStyle.Render("Started: "))
	b.WriteString(detailValueStyle.Render(m.selectedSession.StartedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	if m.selectedSession.CompletedAt != nil {
		b.WriteString(detailLabelStyle.Render("Completed: "))
		b.WriteString(detailValueStyle.Render(m.selectedSession.CompletedAt.Format("2006-01-02 15:04:05")))
		elapsed := m.selectedSession.CompletedAt.Sub(m.selectedSession.StartedAt)
		b.WriteString(detailValueStyle.Render(fmt.Sprintf(" (%s)", elapsed.Round(time.Second))))
		b.WriteString("\n")
	} else {
		elapsed := time.Since(m.selectedSession.StartedAt)
		b.WriteString(detailLabelStyle.Render("Elapsed: "))
		b.WriteString(detailValueStyle.Render(elapsed.Round(time.Second).String()))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Task Progress
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Tasks (%d):", len(m.selectedSession.TaskIDs))))
	b.WriteString("\n")

	if len(m.selectedSession.TaskIDs) == 0 {
		b.WriteString(detailValueStyle.Render("  No tasks in this session"))
		b.WriteString("\n")
	} else {
		for _, taskID := range m.selectedSession.TaskIDs {
			progress, hasProgress := m.selectedSession.TaskProgress[taskID]

			// Task status icon
			var taskIcon string
			var taskColor lipgloss.Color
			if hasProgress {
				switch progress.Status {
				case "completed":
					taskIcon = "✓"
					taskColor = successColor
				case "running":
					taskIcon = "▶"
					taskColor = warningColor
				case "failed":
					taskIcon = "✗"
					taskColor = dangerColor
				default:
					taskIcon = "○"
					taskColor = mutedColor
				}
			} else {
				taskIcon = "○"
				taskColor = mutedColor
			}

			iconPart := lipgloss.NewStyle().Foreground(taskColor).Render(taskIcon)
			b.WriteString(fmt.Sprintf("  %s %s", iconPart, taskID))

			if hasProgress {
				// Show stage information
				if len(progress.Stages) > 0 {
					b.WriteString(detailValueStyle.Render(" — Stages: "))
					stageNames := make([]string, 0, len(progress.Stages))
					for stageName := range progress.Stages {
						stageNames = append(stageNames, stageName)
					}

					for i, stageName := range stageNames {
						stage := progress.Stages[stageName]
						var stageIcon string
						var stageColor lipgloss.Color

						switch stage.Status {
						case "done":
							stageIcon = "✓"
							stageColor = successColor
						case "running":
							stageIcon = "▶"
							stageColor = warningColor
						default:
							stageIcon = "○"
							stageColor = mutedColor
						}

						stageStyled := lipgloss.NewStyle().Foreground(stageColor).Render(
							fmt.Sprintf("%s %s", stageIcon, stageName))

						if stage.Iterations > 1 {
							stageStyled += lipgloss.NewStyle().Foreground(mutedColor).Render(
								fmt.Sprintf("(x%d)", stage.Iterations))
						}

						b.WriteString(stageStyled)
						if i < len(stageNames)-1 {
							b.WriteString(detailValueStyle.Render(" → "))
						}
					}
				}

				// Show elapsed time if completed
				if progress.CompletedAt != nil {
					elapsed := progress.CompletedAt.Sub(progress.StartedAt)
					b.WriteString(detailValueStyle.Render(
						fmt.Sprintf(" [%s]", elapsed.Round(time.Second))))
				}
			}

			b.WriteString("\n")
		}
	}

	content := detailBoxStyle.Width(m.width - 4).Render(b.String())
	help := helpStyle.Render("b/esc: back • r: refresh • q: quit")

	return content + "\n\n" + help
}

// orchestrationStartedMsg is sent when orchestration starts.
type orchestrationStartedMsg struct {
	taskID  string
	success bool
	err     error
}

// startOrchestration starts orchestration for a task in the background.
func (m Model) viewDelegate() string {
	var b strings.Builder

	title := titleStyle.Render("═══ AI Matt Delegation ═══")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.delegateStatus != "" {
		statusStyle := lipgloss.NewStyle().Foreground(successColor)
		b.WriteString(statusStyle.Render(m.delegateStatus))
		b.WriteString("\n\n")
	}

	if m.delegateConfirm {
		confirmStyle := lipgloss.NewStyle().Foreground(warningColor).Bold(true)
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Start supervised delegation with %d interactions? (y/n)", m.delegateInteractions)))
		b.WriteString("\n\n")
		b.WriteString("This will:\n")
		b.WriteString("  • Create a dedicated delegation window\n")
		b.WriteString("  • Start Worker Claude and AI Matt agents\n")
		b.WriteString("  • Add a monitor pane to your current window\n")
		b.WriteString("\n")
	} else {
		b.WriteString("Supervised delegation runs Worker Claude and AI Matt in a\n")
		b.WriteString("dedicated window while you continue working.\n\n")

		b.WriteString("A monitor pane shows communications and approval prompts.\n\n")

		interactionsStyle := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 2)

		b.WriteString("Interactions: ")
		b.WriteString(interactionsStyle.Render(fmt.Sprintf("◀ %d ▶", m.delegateInteractions)))
		b.WriteString("\n\n")

		b.WriteString(helpStyle.Render("↑/↓: adjust • enter: start • q/esc: cancel"))
	}

	return b.String()
}

func (m Model) startOrchestration(taskID string) tea.Cmd {
	return func() tea.Msg {
		// Execute gc orc <task_id> in the background
		cmd := exec.Command("gc", "orc", taskID)
		err := cmd.Start()

		return orchestrationStartedMsg{
			taskID:  taskID,
			success: err == nil,
			err:     err,
		}
	}
}

// Run starts the TUI.
func Run(store *data.Store) error {
	p := tea.NewProgram(New(store), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
