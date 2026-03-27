package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmOpts configures the confirm dialog.
type ConfirmOpts struct {
	// Title is shown bold + yellow after the ⚠ icon.
	Title string
	// Bullets are context lines shown below the title inside the box.
	Bullets []string
}

// ── confirm model ─────────────────────────────────────────────────────────────

type confirmModel struct {
	opts      ConfirmOpts
	confirmed bool
	done      bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(km.String()) {
		case "y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "enter", "ctrl+c", "esc":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	// Build box content
	var lines []string
	lines = append(lines, WarningTitleStyle.Render("⚠  "+m.opts.Title))
	for _, b := range m.opts.Bullets {
		lines = append(lines, "   • "+b)
	}
	boxContent := strings.Join(lines, "\n")
	box := ConfirmBoxStyle.Render(boxContent)

	prompt := fmt.Sprintf("\n  %s %s",
		WarnStyle.Render("确认执行？"),
		GrayStyle.Render("[y/N] ›"),
	)
	return "\n" + box + prompt + "\n"
}

// ── public API ────────────────────────────────────────────────────────────────

// RunConfirm renders a bordered context box and prompts y/N.
// Returns true only if the user presses 'y'.
func RunConfirm(opts ConfirmOpts) bool {
	m, err := tea.NewProgram(confirmModel{opts: opts}).Run()
	if err != nil {
		return false
	}
	return m.(confirmModel).confirmed
}
