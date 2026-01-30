package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	textinputDefaultWidth = 48
	textinputMaxWidth     = 96
	textinputCharLimit    = 1024
)

type textinputPromptModel struct {
	textInput textinput.Model
	question  string
	hint      string
	cancelled bool
	err       error
}

func askSingleLineTextinput(question, placeholder, hint string, input, output *os.File) (string, error) {
	model := newTextinputPromptModel(question, placeholder, hint, output)
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output))
	result, err := program.Run()
	if err != nil {
		return "", err
	}
	m, ok := result.(textinputPromptModel)
	if !ok {
		return "", fmt.Errorf("unexpected textinput model: %T", result)
	}
	if m.cancelled {
		return "", errors.New(i18n.MsgCancelled)
	}
	if m.err != nil {
		return "", m.err
	}
	return strings.TrimSpace(m.textInput.Value()), nil
}

func newTextinputPromptModel(question, placeholder, hint string, output *os.File) textinputPromptModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "  > "
	ti.CharLimit = textinputCharLimit
	ti.Width = textinputWidth(output)
	ti.Focus()

	ti.PromptStyle = StyleMuted
	ti.TextStyle = lipgloss.NewStyle() // default text style
	ti.PlaceholderStyle = StyleMuted

	return textinputPromptModel{
		textInput: ti,
		question:  question,
		hint:      hint,
	}
}

func textinputWidth(output *os.File) int {
	if output == nil {
		return textinputDefaultWidth
	}
	w, _, err := term.GetSize(int(output.Fd()))
	if err != nil || w <= 0 {
		return textinputDefaultWidth
	}
	width := w - 4
	if width < 24 {
		return 24
	}
	if width > textinputMaxWidth {
		return textinputMaxWidth
	}
	return width
}

func (m textinputPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textinputPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg
		return m, nil
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textinputPromptModel) View() string {
	var b strings.Builder
	if m.question != "" {
		b.WriteString(fmt.Sprintf("%s %s\n\n", StyleInfo.Render("?"), m.question))
	}
	b.WriteString(m.textInput.View())
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(StyleError.Render("  " + m.err.Error()))
		b.WriteString("\n")
	}
	if m.hint != "" {
		b.WriteString(StyleMuted.Render("  " + m.hint))
		b.WriteString("\n")
	}
	return b.String()
}
