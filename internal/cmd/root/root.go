// Package root assembles the top-level cobra command tree.
package root

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	fakeapi "github.com/vika2603/100x-cli/api/futures/fake"
	"github.com/vika2603/100x-cli/internal/cmd/completion"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	futuresGroup "github.com/vika2603/100x-cli/internal/cmd/futures"
	"github.com/vika2603/100x-cli/internal/cmd/profile"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/version"
)

type globalFlags struct {
	profile string
	jsonOut bool
	jq      string
	quiet   bool
	yes     bool
	dryRun  bool
	color   string
	timeout time.Duration
}

// ErrorEmitter writes an error to stderr in the format selected by the
// global --json flag: a structured `{"error":{...},"exit_code":N}` JSON
// object for machine consumers, or a plain `error: <msg>` line for
// humans.
type ErrorEmitter func(err error, code int, codeString string)

// NewCmdRoot returns the `100x` cobra command and a function that knows
// how to format end-of-run errors per the active output mode. The
// emitter must be called by main.go once Execute returns an error so
// the format respects --json.
func NewCmdRoot() (*cobra.Command, ErrorEmitter) {
	gf := &globalFlags{}
	f := &factory.Factory{}

	cmd := &cobra.Command{
		Use:           "100x",
		Short:         "100x futures-trading CLI",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&gf.profile, "profile", "", "credential profile to use")
	cmd.PersistentFlags().BoolVar(&gf.jsonOut, "json", false, "emit JSON to stdout")
	cmd.PersistentFlags().StringVar(&gf.jq, "jq", "", "gojq expression applied to JSON output")
	cmd.PersistentFlags().BoolVarP(&gf.quiet, "quiet", "q", false, "suppress non-essential stdout")
	cmd.PersistentFlags().BoolVarP(&gf.yes, "yes", "y", false, "auto-approve destructive prompts")
	cmd.PersistentFlags().BoolVar(&gf.dryRun, "dry-run", false, "show actions without sending")
	cmd.PersistentFlags().StringVar(&gf.color, "color", "auto", "color mode: auto | always | never (NO_COLOR honoured)")
	cmd.PersistentFlags().DurationVar(&gf.timeout, "timeout", 30*time.Second, "per-request HTTP timeout")
	_ = cmd.RegisterFlagCompletionFunc("color", cobra.FixedCompletions([]string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp))

	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		if isCredentialFreeCmd(c) {
			r, err := newRenderer(gf)
			if err != nil {
				return err
			}
			f.IO = r
			f.DryRun = gf.dryRun
			f.Yes = gf.yes
			return nil
		}
		return populate(f, gf)
	}

	cmd.AddCommand(
		futuresGroup.NewCmdFutures(f),
		profile.NewCmdProfile(f),
		completion.NewCmdCompletion(),
	)

	emit := func(err error, code int, codeString string) {
		if gf.jsonOut || gf.jq != "" {
			payload := struct {
				Error    errorPayload `json:"error"`
				ExitCode int          `json:"exit_code"`
			}{
				Error:    errorPayload{Code: codeString, Message: err.Error()},
				ExitCode: code,
			}
			enc := json.NewEncoder(os.Stderr)
			enc.SetIndent("", "  ")
			_ = enc.Encode(payload)
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return cmd, emit
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// isCredentialFreeCmd reports whether c is in a subtree that does not need
// a Factory.Client. profile add / list / use / show / remove and completion
// run before any credentials exist.
func isCredentialFreeCmd(c *cobra.Command) bool {
	for cur := c; cur != nil; cur = cur.Parent() {
		switch cur.Name() {
		case "profile", "completion", "help":
			return true
		}
	}
	return false
}

// populate resolves the profile, secret, and SDK client into f.
func populate(f *factory.Factory, gf *globalFlags) error {
	r, err := newRenderer(gf)
	if err != nil {
		return err
	}
	f.IO = r
	f.DryRun = gf.dryRun
	f.Yes = gf.yes
	f.Timeout = gf.timeout

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if os.Getenv("E100X_FAKE") == "1" {
		f.Client = futures.NewWithDoer(fakeapi.New())
		return nil
	}

	name, p, err := config.Resolve(cfg, gf.profile)
	if err != nil {
		if errors.Is(err, config.ErrNoProfile) {
			return fmt.Errorf("no profile configured: run `100x profile add` or set E100X_FAKE=1")
		}
		return err
	}
	secret, err := credential.Default().Load(name)
	if err != nil {
		return fmt.Errorf("load credentials for profile %q: %w", name, err)
	}
	f.ProfileName = name
	f.Profile = p
	f.Client = futures.New(futures.Options{
		Endpoint:   p.Endpoint,
		ClientID:   p.ClientID,
		ClientKey:  secret,
		HTTPClient: &http.Client{Timeout: gf.timeout},
	})
	return nil
}

func newRenderer(gf *globalFlags) (*output.Renderer, error) {
	r := output.New()
	if gf.jsonOut || gf.jq != "" {
		r.Format = output.FormatJSON
		r.JQ = gf.jq
	}
	r.Quiet = gf.quiet
	mode, err := parseColor(gf.color)
	if err != nil {
		return nil, err
	}
	r.Color = mode
	// Propagate --color never to downstream libs (huh / lipgloss) by setting
	// NO_COLOR; --color always clears it so an explicit override beats the env.
	switch mode {
	case output.ColorNever:
		_ = os.Setenv("NO_COLOR", "1")
	case output.ColorAlways:
		_ = os.Unsetenv("NO_COLOR")
	}
	return r, nil
}

func parseColor(s string) (output.ColorMode, error) {
	switch s {
	case "", "auto":
		return output.ColorAuto, nil
	case "always":
		return output.ColorAlways, nil
	case "never":
		return output.ColorNever, nil
	default:
		return 0, fmt.Errorf("invalid --color %q (want auto, always, never)", s)
	}
}
