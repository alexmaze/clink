package tui

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ErrCancelled is returned when the user presses Esc or Ctrl+C.
var ErrCancelled = errors.New("cancelled")

// ── input model ──────────────────────────────────────────────────────────────

type inputModel struct {
	input     textinput.Model
	label     string
	done      bool
	cancelled bool
}

func newInputModel(label, defaultVal string, masked bool) inputModel {
	ti := textinput.New()
	ti.Focus()
	ti.SetValue(defaultVal)
	ti.Width = 60
	if masked {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	return inputModel{input: ti, label: label}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	prompt := CursorStyle.Render("❯")
	return fmt.Sprintf("\n  %s %s\n  %s %s\n\n",
		WarnStyle.Render(m.label),
		GrayStyle.Render("(Enter to confirm, Esc to cancel)"),
		prompt,
		m.input.View(),
	)
}

// ── public API ───────────────────────────────────────────────────────────────

// RunInput shows a single-line text input prompt.
// Returns (value, nil) on confirm, or ("", ErrCancelled) on cancel.
func RunInput(label, defaultVal string) (string, error) {
	m, err := tea.NewProgram(newInputModel(label, defaultVal, false)).Run()
	if err != nil {
		return "", err
	}
	result := m.(inputModel)
	if result.cancelled {
		return "", ErrCancelled
	}
	return result.input.Value(), nil
}

// RunMaskedInput shows a password input prompt (characters hidden as *).
// Returns (value, nil) on confirm, or ("", ErrCancelled) on cancel.
func RunMaskedInput(label string) (string, error) {
	m, err := tea.NewProgram(newInputModel(label, "", true)).Run()
	if err != nil {
		return "", err
	}
	result := m.(inputModel)
	if result.cancelled {
		return "", ErrCancelled
	}
	return result.input.Value(), nil
}
