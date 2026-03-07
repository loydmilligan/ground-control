// Package tui implements the Bubble Tea TUI for Ground Control.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Question represents a single Q&A prompt.
type Question struct {
	ID          string
	Text        string
	Options     []Option
	AllowOther  bool
	Required    bool
}

// Option represents a single answer choice.
type Option struct {
	Key         string // A, B, C, D
	Label       string // Short label
	Description string // Longer explanation
	Recommended bool   // Show "(Recommended)" tag
}

// Answer represents a user's answer to a question.
type Answer struct {
	QuestionID  string
	SelectedKey string // A, B, C, D, or "other"
	CustomText  string // If "other" selected
	Reasoning   string // Optional explanation
}

// QAModel is the Bubble Tea model for Q&A interaction.
type QAModel struct {
	questions   []Question
	currentIdx  int
	answers     []Answer
	cursor      int  // Which option is highlighted
	inputMode   bool // True when entering custom text
	textInput   textinput.Model
	done        bool
	width       int
	height      int
}

// Styles for QA component
var (
	qaBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	qaQuestionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	qaOptionStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	qaSelectedStyle = lipgloss.NewStyle().
			PaddingLeft(0).
			Bold(true).
			Foreground(primaryColor)

	qaDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			PaddingLeft(7).
			MarginBottom(1)

	qaRecommendedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Italic(true)

	qaHelpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Border(lipgloss.Border{Top: "─"}, false, false, true, false).
			BorderForeground(primaryColor).
			PaddingTop(1)
)

// NewQAModel creates a new Q&A model.
func NewQAModel(questions []Question) QAModel {
	ti := textinput.New()
	ti.Placeholder = "Type your answer..."
	ti.CharLimit = 200
	ti.Width = 50

	return QAModel{
		questions: questions,
		currentIdx: 0,
		answers:    make([]Answer, 0),
		cursor:     0,
		inputMode:  false,
		textInput:  ti,
		done:       false,
	}
}

// Init implements tea.Model.
func (m QAModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m QAModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	// Update text input if in input mode
	if m.inputMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m QAModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle text input mode
	if m.inputMode {
		switch key {
		case "esc":
			m.inputMode = false
			m.textInput.Reset()
			return m, nil
		case "enter":
			// Save custom text answer
			answer := Answer{
				QuestionID:  m.questions[m.currentIdx].ID,
				SelectedKey: "other",
				CustomText:  m.textInput.Value(),
			}
			m.answers = append(m.answers, answer)
			m.textInput.Reset()
			m.inputMode = false
			m.currentIdx++
			m.cursor = 0
			if m.currentIdx >= len(m.questions) {
				m.done = true
			}
			return m, nil
		}
		// Let textinput handle other keys
		return m, nil
	}

	// Handle option selection mode
	question := m.questions[m.currentIdx]

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "j", "down":
		maxCursor := len(question.Options)
		if question.AllowOther {
			maxCursor++
		}
		if m.cursor < maxCursor-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "o":
		// Quick select "Other" option
		if question.AllowOther {
			m.inputMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "tab":
		// Skip to next question if not required
		if !question.Required {
			m.currentIdx++
			m.cursor = 0
			if m.currentIdx >= len(m.questions) {
				m.done = true
			}
		}
		return m, nil

	case "enter":
		// Select current option
		if m.cursor < len(question.Options) {
			// Regular option selected
			opt := question.Options[m.cursor]
			answer := Answer{
				QuestionID:  question.ID,
				SelectedKey: opt.Key,
			}
			m.answers = append(m.answers, answer)
			m.currentIdx++
			m.cursor = 0
			if m.currentIdx >= len(m.questions) {
				m.done = true
			}
		} else if question.AllowOther {
			// "Other" option selected
			m.inputMode = true
			m.textInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "esc":
		// Go back to previous question
		if m.currentIdx > 0 {
			m.currentIdx--
			m.cursor = 0
			// Remove the last answer
			if len(m.answers) > 0 {
				m.answers = m.answers[:len(m.answers)-1]
			}
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
func (m QAModel) View() string {
	if m.done {
		return m.viewDone()
	}

	if m.currentIdx >= len(m.questions) {
		return "No questions available"
	}

	question := m.questions[m.currentIdx]

	var b strings.Builder

	// Question header
	header := fmt.Sprintf("Question %d of %d", m.currentIdx+1, len(m.questions))
	b.WriteString(qaQuestionStyle.Render(header))
	b.WriteString("\n\n")

	// Question text
	b.WriteString(qaQuestionStyle.Render(question.Text))
	b.WriteString("\n\n")

	// Input mode view
	if m.inputMode {
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
	} else {
		// Options
		for i, opt := range question.Options {
			prefix := "  "
			style := qaOptionStyle
			if i == m.cursor {
				prefix = "> "
				style = qaSelectedStyle
			}

			label := fmt.Sprintf("[%s] %s", opt.Key, opt.Label)
			if opt.Recommended {
				label += " " + qaRecommendedStyle.Render("(Recommended)")
			}

			b.WriteString(style.Render(prefix + label))
			b.WriteString("\n")

			if opt.Description != "" {
				b.WriteString(qaDescStyle.Render(opt.Description))
				b.WriteString("\n")
			}
		}

		// "Other" option
		if question.AllowOther {
			otherIdx := len(question.Options)
			prefix := "  "
			style := qaOptionStyle
			if m.cursor == otherIdx {
				prefix = "> "
				style = qaSelectedStyle
			}
			b.WriteString(style.Render(prefix + "[D] Other..."))
			b.WriteString("\n")
		}
	}

	// Help bar
	var helpText string
	if m.inputMode {
		helpText = "Enter confirm • Esc cancel"
	} else {
		parts := []string{"j/k move", "Enter select"}
		if question.AllowOther {
			parts = append(parts, "o other")
		}
		if !question.Required {
			parts = append(parts, "Tab skip")
		}
		if m.currentIdx > 0 {
			parts = append(parts, "Esc back")
		}
		helpText = strings.Join(parts, " • ")
	}

	// Wrap in box
	content := qaBoxStyle.Width(m.width - 4).Render(b.String())
	help := qaHelpBarStyle.Width(m.width - 4).Render(helpText)

	return content + "\n" + help
}

func (m QAModel) viewDone() string {
	var b strings.Builder

	b.WriteString(qaQuestionStyle.Render("Q&A Complete"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Answered %d questions", len(m.answers)))
	b.WriteString("\n\n")

	content := qaBoxStyle.Render(b.String())
	help := qaHelpBarStyle.Render("Press any key to continue")

	return content + "\n" + help
}

// GetAnswers returns all answers collected.
func (m QAModel) GetAnswers() []Answer {
	return m.answers
}

// Done returns true if Q&A is complete.
func (m QAModel) Done() bool {
	return m.done
}
