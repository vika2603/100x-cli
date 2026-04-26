package output

import "github.com/charmbracelet/lipgloss"

// Terminal tone primitives. Call sites decide which tone fits each value
// (for example BUY -> Positive, SELL -> Negative). The Renderer methods
// return ANSI-wrapped text in tty + colour mode and unchanged text otherwise,
// so piped output and --color never stay clean.
var (
	tonePositive = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	tonePending  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	toneNegative = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	toneSubtle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	toneAccent   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// Positive renders s in green when colour is on, otherwise unchanged.
func (r *Renderer) Positive(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return tonePositive.Render(s)
}

// Pending renders s in yellow when colour is on, otherwise unchanged.
func (r *Renderer) Pending(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return tonePending.Render(s)
}

// Negative renders s in red when colour is on, otherwise unchanged.
func (r *Renderer) Negative(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return toneNegative.Render(s)
}

// Subtle renders s in dim gray when colour is on, otherwise unchanged.
func (r *Renderer) Subtle(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return toneSubtle.Render(s)
}

// Accent renders s in cyan when colour is on, otherwise unchanged.
func (r *Renderer) Accent(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return toneAccent.Render(s)
}
