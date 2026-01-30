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

// errMsg wraps errors from the textarea (e.g. paste failures) for handling in Update.
type errMsg error

type textareaPromptModel struct {
	textarea  textarea.Model
	question  string
	hint      string
	cancelled bool
	err       error
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
	model := newTextareaPromptModel(question, i18n.MsgTextareaSubmitHint, i18n.MsgTextareaPlaceholder, output)
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
	if finalModel.err != nil {
		return nil, finalModel.err
	}

	value := finalModel.textarea.Value()
	if value == "" {
		return nil, nil
	}
	return strings.Split(value, "\n"), nil
}

func newTextareaPromptModel(question, hint, placeholder string, output *os.File) textareaPromptModel {
	ti := textarea.New()
	ti.Placeholder = placeholder
	ti.Prompt = "  > "
	ti.ShowLineNumbers = false
	ti.Focus()

	width, height := textareaSize(output)
	ti.SetWidth(width)
	ti.SetHeight(height)

	focusedStyle, blurredStyle := textarea.DefaultStyles()
	focusedStyle.Prompt = StyleMuted
	focusedStyle.Placeholder = StyleMuted
	blurredStyle.Prompt = StyleMuted
	blurredStyle.Placeholder = StyleMuted
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
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		default:
			// Re-focus when blurred (e.g. after Esc) so any key brings focus back
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m textareaPromptModel) View() string {
	var b strings.Builder
	if m.question != "" {
		b.WriteString(fmt.Sprintf("%s %s\n\n", StyleInfo.Render("?"), m.question))
	}
	b.WriteString(m.textarea.View())
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

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
