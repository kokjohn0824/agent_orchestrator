package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const (
	listMinHeight    = 5
	listMaxHeight    = 14
	listDefaultWidth = 48
	listDefaultHeight = 10
)

// selectItem implements list.Item for the selection list.
type selectItem string

func (i selectItem) FilterValue() string { return "" }

// listSelectDelegate renders each list item (list-simple style).
type listSelectDelegate struct{}

func (d listSelectDelegate) Height() int  { return 1 }
func (d listSelectDelegate) Spacing() int { return 0 }
func (d listSelectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d listSelectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(selectItem)
	if !ok {
		return
	}
	str := fmt.Sprintf("%d. %s", index+1, string(item))
	if index == m.Index() {
		fmt.Fprint(w, StylePrimary.Render("> "+str))
	} else {
		fmt.Fprint(w, StyleMuted.Render(str))
	}
}

type listSelectModel struct {
	list       list.Model
	title      string
	choiceIndex int // -1 = not chosen yet
	cancelled  bool
}

func askSelectList(question string, options []string, input, output *os.File) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("no options to select")
	}
	model := newListSelectModel(question, options, output)
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output))
	result, err := program.Run()
	if err != nil {
		return 0, err
	}
	m, ok := result.(listSelectModel)
	if !ok {
		return 0, fmt.Errorf("unexpected list select model: %T", result)
	}
	if m.cancelled {
		return 0, errors.New(i18n.MsgCancelled)
	}
	return m.choiceIndex, nil
}

func newListSelectModel(question string, options []string, output *os.File) listSelectModel {
	items := make([]list.Item, len(options))
	for i, s := range options {
		items[i] = selectItem(s)
	}
	width := listDefaultWidth
	height := listDefaultHeight
	if output != nil {
		if w, h, err := term.GetSize(int(output.Fd())); err == nil && w > 0 && h > 0 {
			width = w - 4
			if width < 24 {
				width = 24
			}
			height = h / 2
			if height < listMinHeight {
				height = listMinHeight
			}
			if height > listMaxHeight {
				height = listMaxHeight
			}
		}
	}
	l := list.New(items, listSelectDelegate{}, width, height)
	l.Title = question
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = StyleTitle
	l.Styles.PaginationStyle = StyleMuted
	l.Styles.HelpStyle = StyleMuted
	return listSelectModel{
		list:        l,
		title:      question,
		choiceIndex: -1,
	}
}

func (m listSelectModel) Init() tea.Cmd {
	return nil
}

func (m listSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.choiceIndex = m.list.Index()
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listSelectModel) View() string {
	if m.choiceIndex >= 0 {
		return ""
	}
	if m.cancelled {
		return StyleMuted.Render("  " + i18n.MsgCancelled) + "\n"
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.list.View())
	return b.String()
}
