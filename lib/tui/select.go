package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ErrNoSelection is returned when the user cancels the selection.
var ErrNoSelection = errors.New("no selection")

// ── select model ──────────────────────────────────────────────────────────────

type selectModel struct {
	label     string
	items     []string
	cursor    int
	selected  int
	cancelled bool
	// visible window
	windowSize int
	offset     int
}

func newSelectModel(label string, items []string) selectModel {
	size := 10
	if len(items) < size {
		size = len(items)
	}
	return selectModel{
		label:      label,
		items:      items,
		selected:   -1,
		windowSize: size,
	}
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case tea.KeyDown:
			if m.cursor < len(m.items)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.windowSize {
					m.offset = m.cursor - m.windowSize + 1
				}
			}
		case tea.KeyEnter:
			m.selected = m.cursor
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyRunes:
			if km.String() == "q" {
				m.cancelled = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  %s\n\n", WarnStyle.Render(m.label)))

	end := m.offset + m.windowSize
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := m.offset; i < end; i++ {
		item := m.items[i]
		if i == m.cursor {
			sb.WriteString(fmt.Sprintf("  %s %s\n",
				CursorStyle.Render("❯"),
				SelectedStyle.Render(item),
			))
		} else {
			sb.WriteString(fmt.Sprintf("    %s\n", GrayStyle.Render(item)))
		}
	}

	if len(m.items) > m.windowSize {
		shown := m.offset + m.windowSize
		if shown > len(m.items) {
			shown = len(m.items)
		}
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			GrayStyle.Render(fmt.Sprintf("(%d/%d  ↑↓ to navigate, Enter to confirm, Esc to cancel)",
				m.cursor+1, len(m.items))),
		))
	} else {
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			GrayStyle.Render("(↑↓ to navigate, Enter to confirm, Esc to cancel)"),
		))
	}
	return sb.String()
}

// ── public API ────────────────────────────────────────────────────────────────

// RunSelect shows a scrollable list and returns the selected index.
// Returns (-1, ErrNoSelection) if the user cancels.
func RunSelect(label string, items []string) (int, error) {
	m, err := tea.NewProgram(newSelectModel(label, items)).Run()
	if err != nil {
		return -1, err
	}
	result := m.(selectModel)
	if result.cancelled {
		return -1, ErrNoSelection
	}
	return result.selected, nil
}
