package output

import "github.com/charmbracelet/lipgloss"

// Semantic colour primitives. Call sites decide which one fits each
// value (e.g. trade-side BUY → Success, SELL → Danger). The Renderer
// methods below return the ANSI-wrapped form in tty + colour mode and
// the input unchanged otherwise, so the same call site code path
// produces clean text in piped / JSON / --no-color contexts.
var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleDanger  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// Success renders s in green when colour is on, otherwise unchanged.
func (r *Renderer) Success(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return styleSuccess.Render(s)
}

// Warning renders s in yellow when colour is on, otherwise unchanged.
func (r *Renderer) Warning(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return styleWarning.Render(s)
}

// Danger renders s in red when colour is on, otherwise unchanged.
func (r *Renderer) Danger(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return styleDanger.Render(s)
}

// Muted renders s in dim gray when colour is on, otherwise unchanged.
func (r *Renderer) Muted(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return styleMuted.Render(s)
}

// Info renders s in cyan when colour is on, otherwise unchanged.
func (r *Renderer) Info(s string) string {
	if !r.ColorOnStdout() {
		return s
	}
	return styleInfo.Render(s)
}
