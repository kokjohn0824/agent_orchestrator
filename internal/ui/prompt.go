package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
)

// Prompt handles interactive user prompts
type Prompt struct {
	reader io.Reader
	writer io.Writer
}

// NewPrompt creates a new prompt handler
func NewPrompt(r io.Reader, w io.Writer) *Prompt {
	return &Prompt{
		reader: r,
		writer: w,
	}
}

// Ask asks a question and returns the user's answer
func (p *Prompt) Ask(question string) (string, error) {
	fmt.Fprintf(p.writer, "%s %s\n", StyleInfo.Render("?"), question)
	fmt.Fprint(p.writer, StyleMuted.Render("  > "))

	scanner := bufio.NewScanner(p.reader)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// AskMultiline asks a question that can have multiple lines of input
func (p *Prompt) AskMultiline(question string) ([]string, error) {
	if input, output, ok := canUseTextarea(p.reader, p.writer); ok {
		return askMultilineTextarea(question, input, output)
	}

	return p.askMultilineScanner(question)
}

func (p *Prompt) askMultilineScanner(question string) ([]string, error) {
	fmt.Fprintf(p.writer, "%s %s\n", StyleInfo.Render("?"), question)
	fmt.Fprintln(p.writer, StyleMuted.Render("  "+i18n.MsgInputEndHint))

	var lines []string
	scanner := bufio.NewScanner(p.reader)

	for {
		fmt.Fprint(p.writer, StyleMuted.Render("  > "))
		if !scanner.Scan() {
			break
		}
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// Confirm asks a yes/no question
func (p *Prompt) Confirm(question string, defaultYes bool) (bool, error) {
	defaultHint := "[y/N]"
	if defaultYes {
		defaultHint = "[Y/n]"
	}

	fmt.Fprintf(p.writer, "%s %s %s ", StyleInfo.Render("?"), question, StyleMuted.Render(defaultHint))

	scanner := bufio.NewScanner(p.reader)
	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch answer {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		case "":
			return defaultYes, nil
		default:
			return defaultYes, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return defaultYes, nil
}

// Select asks the user to select from a list of options.
// When running in a TTY, uses an interactive list (bubbletea list-simple style);
// otherwise falls back to numbered prompt + scanner input.
func (p *Prompt) Select(question string, options []string) (int, error) {
	if input, output, ok := canUseTextarea(p.reader, p.writer); ok {
		return askSelectList(question, options, input, output)
	}
	return p.askSelectScanner(question, options)
}

func (p *Prompt) askSelectScanner(question string, options []string) (int, error) {
	fmt.Fprintf(p.writer, "%s %s\n", StyleInfo.Render("?"), question)
	for i, opt := range options {
		fmt.Fprintf(p.writer, "  %s %s\n", StyleMuted.Render(fmt.Sprintf("%d.", i+1)), opt)
	}
	fmt.Fprint(p.writer, StyleMuted.Render(fmt.Sprintf("  "+i18n.MsgSelectRange, len(options))))

	scanner := bufio.NewScanner(p.reader)
	if scanner.Scan() {
		answer := strings.TrimSpace(scanner.Text())
		var choice int
		if _, err := fmt.Sscanf(answer, "%d", &choice); err == nil {
			if choice >= 1 && choice <= len(options) {
				return choice - 1, nil
			}
		}
		return 0, fmt.Errorf(i18n.MsgInvalidSelection, answer)
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, io.EOF
}

// Print helpers

// PrintHeader prints a styled header
func PrintHeader(w io.Writer, title string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, StyleTitle.Render(title))
}

// PrintSubheader prints a styled subheader
func PrintSubheader(w io.Writer, title string) {
	fmt.Fprintln(w, StyleSubtitle.Render(title))
}

// PrintSuccess prints a success message
func PrintSuccess(w io.Writer, message string) {
	fmt.Fprintf(w, "%s %s\n", StyleSuccess.Render("✓"), message)
}

// PrintError prints an error message
func PrintError(w io.Writer, message string) {
	fmt.Fprintf(w, "%s %s\n", StyleError.Render("✗"), message)
}

// PrintWarning prints a warning message
func PrintWarning(w io.Writer, message string) {
	fmt.Fprintf(w, "%s %s\n", StyleWarning.Render("!"), message)
}

// PrintInfo prints an info message
func PrintInfo(w io.Writer, message string) {
	fmt.Fprintf(w, "%s %s\n", StyleInfo.Render("ℹ"), message)
}

// PrintStep prints a step indicator
func PrintStep(w io.Writer, current, total int, message string) {
	step := StylePrimary.Render(fmt.Sprintf("[%d/%d]", current, total))
	fmt.Fprintf(w, "%s %s\n", step, message)
}
