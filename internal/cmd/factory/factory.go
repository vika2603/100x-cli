// Package factory wires the dependencies every command needs and injects
// them by constructor rather than via globals or context-stuffing.
//
// A Factory is built once in cmd/100x/main.go (or in tests) and passed into
// each NewCmdXxx(f *Factory). Subcommands close over the Factory; no global
// state and no smuggling-through-cobra.Context.
package factory

import (
	"errors"
	"fmt"
	"time"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/prompt"
	"github.com/vika2603/100x-cli/internal/session"
)

// Factory carries the live dependencies for a CLI invocation.
type Factory struct {
	// IO is the output renderer (stdout / stderr / format / jq / quiet).
	IO *output.Renderer

	// Yes is surfaced for verbs that prompt before destructive actions.
	Yes bool

	// Timeout caps each HTTP request the SDK makes. Zero means use the
	// SDK default.
	Timeout time.Duration

	// Auth is the resolved auth mode for the current verb. Captured by root
	// PersistentPreRunE from the cobra annotation tree.
	Auth AuthMode

	// ProfileFlag is the raw --profile flag value (empty falls back to
	// E100X_PROFILE / Config.Default).
	ProfileFlag string

	// Lazy state. Populated on first call to Futures() / ProfileName(). The
	// keychain read for AuthPrivate happens only inside Futures() so a verb
	// that errors out at flag validation never triggers it.
	futures       *futures.Client
	futuresErr    error
	futuresLoaded bool

	profileName   string
	profileLoaded bool
}

// Futures returns the API client for the active auth mode. The first call
// performs the load: config + endpoint for AuthPublic; same plus the
// keychain secret read for AuthPrivate. Subsequent calls return the cached
// client (or the cached error).
func (f *Factory) Futures() (*futures.Client, error) {
	if !f.futuresLoaded {
		f.futuresLoaded = true
		sess, err := session.Load(session.LoadOptions{
			RequestedProfile: f.ProfileFlag,
			Timeout:          f.Timeout,
			Public:           f.Auth == AuthPublic,
		})
		if err != nil {
			f.futuresErr = friendlySessionErr(err)
			return nil, f.futuresErr
		}
		f.futures = sess.Client
		f.profileName = sess.ProfileName
		f.profileLoaded = true
	}
	return f.futures, f.futuresErr
}

// ProfileName returns the resolved active-profile name, or "" for public /
// no-auth flows. Cheap: never reads the keychain. The first call resolves
// against config.toml; if Futures() ran first, the cached name is reused.
func (f *Factory) ProfileName() string {
	if !f.profileLoaded {
		f.profileLoaded = true
		if f.Auth == AuthPrivate {
			if cfg, err := config.Load(); err == nil {
				if name, _, err := config.Resolve(cfg, f.ProfileFlag); err == nil {
					f.profileName = name
				}
			}
		}
	}
	return f.profileName
}

// ConfirmDestructive prompts the user before a destructive trading action
// and prefixes the title with the active profile so it's always obvious
// which account the action would hit. Defers to prompt.ConfirmDestructive
// for the four-quadrant tty/--yes matrix.
func (f *Factory) ConfirmDestructive(title string) (bool, error) {
	if name := f.ProfileName(); name != "" {
		title = fmt.Sprintf("[%s] %s", name, title)
	}
	return prompt.ConfirmDestructive(title, f.Yes)
}

// NewForTest returns a Factory pre-wired with a futures client and renderer.
// Pass nil for io to use output.New() defaults. Production code never calls
// this; production builds the Factory empty in root.go and fills inputs at
// PersistentPreRunE.
func NewForTest(c *futures.Client, io *output.Renderer) *Factory {
	if io == nil {
		io = output.New()
	}
	return &Factory{
		IO:            io,
		futures:       c,
		futuresLoaded: true,
		profileLoaded: true,
	}
}

// friendlySessionErr maps the well-known config errors to the wording the
// PersistentPreRunE hook used to produce, now that the load happens at the
// verb. Other errors pass through unchanged.
func friendlySessionErr(err error) error {
	switch {
	case errors.Is(err, config.ErrNoProfile):
		return errors.New("no profile configured: run `100x profile add`")
	case errors.Is(err, config.ErrNoEndpoint):
		return errors.New("no API endpoint configured: set $E100X_ENDPOINT")
	}
	return err
}
