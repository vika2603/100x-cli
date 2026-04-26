// Package prompt centralises interactive user input via charmbracelet/huh.
//
// All interactive prompts in the CLI go through this package so that:
//   - the prompt look and feel is consistent
//   - non-tty contexts can be detected in one place
//   - swapping the underlying library later is a one-file change
package prompt

import (
	"errors"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
)

// ErrNoTTY is returned when an interactive prompt is requested but no
// terminal is attached. Callers should fail with a "pass --yes" hint.
var ErrNoTTY = errors.New("interactive prompt requires a tty (pass -y or --secret)")

// ErrDestructiveNoTTY is returned by ConfirmDestructive when the caller
// is in non-tty mode and -y was not passed. cmd/100x maps it to exit 73.
var ErrDestructiveNoTTY = errors.New("destructive op refused: non-tty without -y")

// ConfirmDestructive implements the four-quadrant tty/-y matrix for
// high-risk operations (bulk cancel, profile remove, …). Pass yes=true
// when the caller's --yes flag is set.
//
//	tty + yes=true  → returns (true, nil) without prompting
//	tty + yes=false → prompts; default answer is "no"
//	!tty + yes=true → returns (true, nil) without prompting
//	!tty + yes=false → returns (false, ErrDestructiveNoTTY)
func ConfirmDestructive(title string, yes bool) (bool, error) {
	if yes {
		return true, nil
	}
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return false, ErrDestructiveNoTTY
	}
	v := false
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&v).
		Run()
	return v, err
}

// Secret reads a single line of input with the screen masked.
//
// Use for API secrets, passphrases, and any other value the user must not
// see in their terminal scrollback.
func Secret(title string) (string, error) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return "", ErrNoTTY
	}
	var v string
	err := huh.NewInput().
		Title(title).
		EchoMode(huh.EchoModePassword).
		Value(&v).
		Run()
	return v, err
}

// Input reads one line of plain text input.
func Input(title, placeholder string) (string, error) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return "", ErrNoTTY
	}
	var v string
	err := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&v).
		Run()
	return v, err
}

// Confirm asks a yes/no question.
//
// Returns def directly in non-tty contexts so callers do not have to special-
// case CI / pipes; pair with --yes / --dry-run flags to keep behaviour
// predictable in scripts.
func Confirm(title string, def bool) (bool, error) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return def, nil
	}
	v := def
	err := huh.NewConfirm().
		Title(title).
		Value(&v).
		Run()
	return v, err
}

// Select presents a list of options and returns the chosen one.
func Select(title string, options []string) (string, error) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return "", ErrNoTTY
	}
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	var v string
	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&v).
		Run()
	return v, err
}
