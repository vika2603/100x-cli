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
	"github.com/charmbracelet/lipgloss"
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
		WithTheme(dangerTheme()).
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
		WithTheme(neutralTheme()).
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
		WithTheme(neutralTheme()).
		Run()
	return v, err
}

// Confirm asks a yes/no question.
//
// Returns def directly in non-tty contexts so callers do not have to special-
// case CI / pipes; pair with --yes where needed to keep behaviour predictable
// in scripts.
func Confirm(title string, def bool) (bool, error) {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return def, nil
	}
	v := def
	err := huh.NewConfirm().
		Title(title).
		Value(&v).
		WithTheme(neutralTheme()).
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
		WithTheme(neutralTheme()).
		Run()
	return v, err
}

// Tokyo Night palette (Night variant). Single source of truth so neutral and
// danger themes stay coherent.
var (
	tnBgDark   = lipgloss.Color("#16161e")
	tnBgHighlt = lipgloss.Color("#292e42")
	tnFg       = lipgloss.Color("#c0caf5")
	tnComment  = lipgloss.Color("#565f89")
	tnBlue     = lipgloss.Color("#7aa2f7")
	tnOrange   = lipgloss.Color("#ff9e64")
	tnRed      = lipgloss.Color("#f7768e")
)

// neutralTheme is the calm default for non-destructive prompts.
// Used by Confirm / Input / Secret / Select.
func neutralTheme() *huh.Theme {
	t := huh.ThemeBase()

	var (
		fg     = tnFg
		muted  = tnComment
		title  = tnFg
		accent = tnBlue
		btnBg  = tnBgHighlt
		errFg  = tnRed
	)

	button := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)

	t.Focused.Base = lipgloss.NewStyle()
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(title).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(title).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(errFg)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(errFg)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(accent)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(accent).SetString("✓ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("• ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.FocusedButton = button.Foreground(tnBgDark).Background(accent).Bold(true)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = button.Foreground(muted).Background(btnBg)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(accent)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(accent)

	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle()
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}

// dangerTheme is reserved for ConfirmDestructive: red bold title plus an
// amber focused button so the prompt reads as "warning" regardless of which
// answer currently has focus.
func dangerTheme() *huh.Theme {
	t := neutralTheme()

	button := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)

	t.Focused.Title = t.Focused.Title.Foreground(tnRed).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(tnRed).Bold(true)
	t.Focused.FocusedButton = button.Foreground(tnBgDark).Background(tnOrange).Bold(true)
	t.Focused.Next = t.Focused.FocusedButton

	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle()
	t.Blurred.Card = t.Blurred.Base

	t.Group.Title = t.Focused.Title
	return t
}
