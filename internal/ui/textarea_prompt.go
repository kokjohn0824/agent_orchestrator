package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const (
	textareaMinWidth      = 40
	textareaMaxWidth      = 96
	textareaMinHeight     = 4
	textareaMaxHeight     = 12
	textareaDefaultWidth  = 80
	textareaDefaultHeight = 8
)

type textareaPromptModel struct {
	textarea  textarea.Model
	question  string
	hint      string
	cancelled bool
}

func canUseTextarea(reader io.Reader, writer io.Writer) (*os.File, *os.File, bool) {
	input, okInput := reader.(*os.File)
	output, okOutput := writer.(*os.File)
	if !okInput || !okOutput {
		return nil, nil, false
	}
	if !term.IsTerminal(int(input.Fd())) || !term.IsTerminal(int(output.Fd())) {
		return nil, nil, false
	}
	return input, output, true
}

func askMultilineTextarea(question string, input, output *os.File) ([]string, error) {
	model := newTextareaPromptModel(question, i18n.MsgTextareaSubmitHint, output)
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output))
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	finalModel, ok := result.(textareaPromptModel)
	if !ok {
		return nil, fmt.Errorf("unexpected textarea model: %T", result)
	}
	if finalModel.cancelled {
		return nil, errors.New(i18n.MsgCancelled)
	}

	value := finalModel.textarea.Value()
	if value == "" {
		return nil, nil
	}
	return strings.Split(value, "\n"), nil
}

func newTextareaPromptModel(question, hint string, output *os.File) textareaPromptModel {
	ti := textarea.New()
	ti.Focus()
	ti.Prompt = "  > "
	ti.ShowLineNumbers = false

	width, height := textareaSize(output)
	ti.SetWidth(width)
	ti.SetHeight(height)

	focusedStyle, blurredStyle := textarea.DefaultStyles()
	focusedStyle.Prompt = StyleMuted
	blurredStyle.Prompt = StyleMuted
	ti.FocusedStyle = focusedStyle
	ti.BlurredStyle = blurredStyle

	return textareaPromptModel{
		textarea: ti,
		question: question,
		hint:     hint,
	}
}

func textareaSize(output *os.File) (int, int) {
	width := textareaDefaultWidth
	height := textareaDefaultHeight
	if output == nil {
		return width, height
	}

	termWidth, termHeight, err := term.GetSize(int(output.Fd()))
	if err != nil || termWidth <= 0 || termHeight <= 0 {
		return width, height
	}

	availableWidth := termWidth - 4
	if availableWidth <= 0 {
		availableWidth = termWidth
	}
	if availableWidth < textareaMinWidth {
		width = availableWidth
	} else {
		width = clampInt(availableWidth, textareaMinWidth, textareaMaxWidth)
	}
	if width <= 0 {
		width = textareaDefaultWidth
	}

	height = clampInt(termHeight/3, textareaMinHeight, textareaMaxHeight)
	if height <= 0 {
		height = textareaDefaultHeight
	}

	return width, height
}

func (m textareaPromptModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m textareaPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textareaPromptModel) View() string {
	var b strings.Builder
	if m.question != "" {
		b.WriteString(fmt.Sprintf("%s %s\n\n", StyleInfo.Render("?"), m.question))
	}
	b.WriteString(m.textarea.View())
	b.WriteString("\n")
	if m.hint != "" {
		b.WriteString(StyleMuted.Render("  " + m.hint))
		b.WriteString("\n")
	}
	return b.String()
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
