package ui

import (
	"errors"

	"github.com/alexmaze/clink/lib/tui"
)

var ErrPromptUnavailable = errors.New("prompt unavailable in non-interactive mode")

type UI interface {
	Confirm(title string, bullets []string) (bool, error)
	Select(title string, items []string) (int, error)
	Input(label, defaultValue string) (string, error)
	Secret(label string) (string, error)
	Interactive() bool
}

type TUI struct{}

func (TUI) Confirm(title string, bullets []string) (bool, error) {
	return tui.RunConfirm(tui.ConfirmOpts{Title: title, Bullets: bullets}), nil
}

func (TUI) Select(title string, items []string) (int, error) {
	return tui.RunSelect(title, items)
}

func (TUI) Input(label, defaultValue string) (string, error) {
	return tui.RunInput(label, defaultValue)
}

func (TUI) Secret(label string) (string, error) {
	return tui.RunMaskedInput(label)
}

func (TUI) Interactive() bool {
	return true
}

type NonInteractive struct{}

func (NonInteractive) Confirm(string, []string) (bool, error) {
	return false, ErrPromptUnavailable
}

func (NonInteractive) Select(string, []string) (int, error) {
	return -1, ErrPromptUnavailable
}

func (NonInteractive) Input(string, string) (string, error) {
	return "", ErrPromptUnavailable
}

func (NonInteractive) Secret(string) (string, error) {
	return "", ErrPromptUnavailable
}

func (NonInteractive) Interactive() bool {
	return false
}
