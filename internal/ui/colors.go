// Package ui provides terminal UI components and styling
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("39")  // Blue
	ColorSuccess   = lipgloss.Color("82")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorInfo      = lipgloss.Color("87")  // Cyan
	ColorMuted     = lipgloss.Color("245") // Gray
	ColorHighlight = lipgloss.Color("212") // Pink
)

// Text styles
var (
	StyleBold = lipgloss.NewStyle().Bold(true)

	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorInfo)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)
)

// Status indicators
var (
	StatusPending    = StyleWarning.Render("○")
	StatusInProgress = StyleInfo.Render("◐")
	StatusCompleted  = StyleSuccess.Render("●")
	StatusFailed     = StyleError.Render("✗")
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Width(60)
)

// Priority styles
func PriorityStyle(priority int) lipgloss.Style {
	switch {
	case priority <= 1:
		return lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	case priority <= 2:
		return lipgloss.NewStyle().Foreground(ColorWarning)
	case priority <= 3:
		return lipgloss.NewStyle().Foreground(ColorInfo)
	default:
		return lipgloss.NewStyle().Foreground(ColorMuted)
	}
}

// Severity styles for analyze command
func SeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "HIGH":
		return lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	case "MED", "MEDIUM":
		return lipgloss.NewStyle().Foreground(ColorWarning)
	case "LOW":
		return lipgloss.NewStyle().Foreground(ColorInfo)
	default:
		return lipgloss.NewStyle().Foreground(ColorMuted)
	}
}
