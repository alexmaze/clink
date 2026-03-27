package tui

import "github.com/charmbracelet/lipgloss"

var (
	// SpinnerStyle is the cyan style used to colour spinner frames.
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // bright cyan

	// ConfirmBoxStyle draws a rounded border in amber/yellow.
	ConfirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")). // amber
			Padding(0, 2)

	// WarningTitleStyle is bold amber, used for the ⚠ title inside confirm box.
	WarningTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214"))

	// HeaderStyle is bold cyan, used for table header cells.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))

	// OKStyle is green text.
	OKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// ErrStyle is red text.
	ErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// WarnStyle is yellow text.
	WarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	// GrayStyle is dim gray text.
	GrayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// CursorStyle is the ❯ prompt cursor color.
	CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)

	// SelectedStyle is the highlighted list item.
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
)
