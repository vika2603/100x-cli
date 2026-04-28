// Package factory wires the dependencies every command needs and injects
// them by constructor rather than via globals or context-stuffing.
//
// A Factory is built once in cmd/100x/main.go (or in tests) and passed into
// each NewCmdXxx(f *Factory). Subcommands close over the Factory; no global
// state and no smuggling-through-cobra.Context.
package factory

import (
	"fmt"
	"time"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/prompt"
)

// Factory carries the live dependencies for a CLI invocation.
type Factory struct {
	// Client is the futures API client. Verbs do not care whether it is a
	// signed HTTP client or a test-injected Doer.
	Client *futures.Client

	// IO is the output renderer (stdout / stderr / format / jq / quiet).
	IO *output.Renderer

	// Profile is the resolved profile metadata, or nil for public-only flows.
	Profile     *config.Profile
	ProfileName string

	// Yes is surfaced for verbs that prompt before destructive actions.
	Yes bool

	// Timeout caps each HTTP request the SDK makes. Zero means use the
	// SDK default.
	Timeout time.Duration
}

// ConfirmDestructive prompts the user before a destructive trading action
// and prefixes the title with the active profile so it's always obvious
// which account the action would hit. Defers to prompt.ConfirmDestructive
// for the four-quadrant tty/--yes matrix.
func (f *Factory) ConfirmDestructive(title string) (bool, error) {
	if f.ProfileName != "" {
		title = fmt.Sprintf("[%s] %s", f.ProfileName, title)
	}
	return prompt.ConfirmDestructive(title, f.Yes)
}
