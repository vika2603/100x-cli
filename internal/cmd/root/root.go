// Package root assembles the top-level cobra command tree.
package root

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/completion"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	futuresGroup "github.com/vika2603/100x-cli/internal/cmd/futures"
	"github.com/vika2603/100x-cli/internal/cmd/profile"
	"github.com/vika2603/100x-cli/internal/cmd/upgrade"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/session"
	"github.com/vika2603/100x-cli/internal/version"
)

type globalFlags struct {
	profile string
	jsonOut bool
	jq      string
	quiet   bool
	yes     bool
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
	cobra.EnableCommandSorting = false

	gf := &globalFlags{}
	f := &factory.Factory{}

	cmd := &cobra.Command{
		Use:   "100x",
		Short: "Command-line interface for the 100X exchange.",
		Example: "# Add a profile named test (you will be prompted for the secret)\n" +
			"  100x profile add test --client-id <CID>\n\n" +
			"# Run one command against a different API host\n" +
			"  E100X_ENDPOINT=https://api.example.com 100x futures market state BTCUSDT\n\n" +
			"# Show the latest ticker-style state for BTCUSDT\n" +
			"  100x futures market state BTCUSDT",
		SilenceUsage:               true,
		SilenceErrors:              true,
		SuggestionsMinimumDistance: 2,
		RunE: func(c *cobra.Command, args []string) error {
			if err := cobra.NoArgs(c, args); err != nil {
				return clierr.Usage(clierr.WithHelpHint(err, c.CommandPath()))
			}
			return c.Help()
		},
	}
	cmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands"},
		&cobra.Group{ID: "auth", Title: "Auth Commands"},
		&cobra.Group{ID: "tools", Title: "Tooling Commands"},
	)

	// CLI takes no filesystem paths anywhere, so flags without an explicit
	// completion function should not fall back to file completion. The
	// directive is inherited by every descendant.
	cmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)

	cmd.PersistentFlags().StringVar(&gf.profile, "profile", "", "Use credentials from profile <name>")
	cmd.PersistentFlags().BoolVar(&gf.jsonOut, "json", false, "Write JSON to stdout")
	cmd.PersistentFlags().StringVar(&gf.jq, "jq", "", "Run a gojq expression against JSON output")
	cmd.PersistentFlags().BoolVarP(&gf.quiet, "quiet", "q", false, "Hide human-readable stdout")
	cmd.PersistentFlags().BoolVarP(&gf.yes, "yes", "y", false, "Answer yes to confirmation prompts")
	cmd.PersistentFlags().StringVar(&gf.color, "color", "auto", "Color mode: auto | always | never (NO_COLOR honored)")
	cmd.PersistentFlags().DurationVar(&gf.timeout, "timeout", 15*time.Second, "HTTP timeout per request")
	_ = cmd.RegisterFlagCompletionFunc("profile", profile.CompleteNameFlag)
	_ = cmd.RegisterFlagCompletionFunc("color", cobra.FixedCompletions([]string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp))

	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		r, err := newRenderer(gf)
		if err != nil {
			return err
		}
		f.IO = r
		f.Yes = gf.yes
		f.Timeout = gf.timeout

		// --timeout bounds the whole command, including retries.
		if gf.timeout > 0 {
			ctx, cancel := context.WithTimeout(c.Context(), gf.timeout) //nolint:gosec // cancel runs via cobra.OnFinalize below
			c.SetContext(ctx)
			cobra.OnFinalize(cancel)
		}

		// Group commands display help; their RunE only calls c.Help(), which
		// never touches the API. Skip client load structurally rather than
		// asking each group to declare AuthNone.
		if c.HasSubCommands() {
			return nil
		}

		// Each verb (or its nearest ancestor) declares its own client need via
		// factory.RequirePublic / RequirePrivate. Unmarked verbs (version,
		// completion, profile management) load nothing.
		mode := factory.LookupAuth(c)
		if mode == factory.AuthNone {
			return nil
		}

		sess, err := session.Load(session.LoadOptions{
			RequestedProfile: gf.profile,
			Timeout:          gf.timeout,
			Public:           mode == factory.AuthPublic,
		})
		if err != nil {
			if errors.Is(err, config.ErrNoProfile) {
				return fmt.Errorf("no profile configured: run `100x profile add`")
			}
			if errors.Is(err, config.ErrNoEndpoint) {
				return fmt.Errorf("no API endpoint configured: set $E100X_ENDPOINT")
			}
			return err
		}
		f.Client = sess.Client
		f.Profile = sess.Profile
		f.ProfileName = sess.ProfileName
		return nil
	}

	futuresCmd := futuresGroup.NewCmdFutures(f)
	futuresCmd.GroupID = "core"
	profileCmd := profile.NewCmdProfile(f)
	profileCmd.GroupID = "auth"
	completionCmd := completion.NewCmdCompletion()
	completionCmd.GroupID = "tools"
	versionCmd := newCmdVersion(f)
	versionCmd.GroupID = "tools"
	upgradeCmd := upgrade.NewCmdUpgrade(f)
	upgradeCmd.GroupID = "tools"
	cmd.AddCommand(futuresCmd, profileCmd, completionCmd, versionCmd, upgradeCmd)
	configureHelp(cmd)
	configureUsageErrors(cmd)

	emit := func(err error, _ int, codeString string) {
		if gf.jsonOut || gf.jq != "" {
			message := addGenericUsageHint(err.Error(), codeString)
			payload := struct {
				Error errorPayload `json:"error"`
			}{Error: errorPayload{Code: codeString, Message: message}}
			var apiErr *futures.APIError
			if errors.As(err, &apiErr) {
				payload.Error.Message = apiErr.Message
				payload.Error.HTTPStatus = apiErr.Status
				payload.Error.APICode = apiErr.Code
			}
			enc := json.NewEncoder(os.Stderr)
			enc.SetIndent("", "  ")
			_ = enc.Encode(payload)
			return
		}
		fmt.Fprintln(os.Stderr, "error:", addGenericUsageHint(humanErrorMessage(err, codeString), codeString))
	}
	return cmd, emit
}

type errorPayload struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
	APICode    int    `json:"api_code,omitempty"`
}

func humanErrorMessage(err error, codeString string) string {
	var apiErr *futures.APIError
	if errors.As(err, &apiErr) {
		return err.Error()
	}
	if codeString != "network" && codeString != "server" {
		return err.Error()
	}
	return summarizeNetworkError(err)
}

func addGenericUsageHint(msg, codeString string) string {
	if codeString != "usage" || strings.Contains(msg, "--help") {
		return msg
	}
	msg = strings.TrimRight(msg, " \t\r\n.")
	if strings.Contains(msg, "\n") {
		return msg + "\nRun `100x --help` for usage"
	}
	return msg + ". Run `100x --help` for usage"
}

func summarizeNetworkError(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return summarizeNetworkCause(urlErr.Err)
	}
	return summarizeNetworkCause(err)
}

func summarizeNetworkCause(err error) string {
	if err == nil {
		return "network error while contacting 100x API"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "tls handshake timeout"):
		return "TLS handshake timed out while connecting to 100x API; retry or increase --timeout"
	case errors.Is(err, os.ErrDeadlineExceeded), errors.Is(err, net.ErrClosed):
		return "network timeout while contacting 100x API; retry or increase --timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "network timeout while contacting 100x API; retry or increase --timeout"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "DNS lookup failed for 100x API endpoint"
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return "connection failed while contacting 100x API"
		}
	}
	return "network error while contacting 100x API"
}

type versionPayload struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func newCmdVersion(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Example: "# Print the CLI version, commit, and build date\n" +
			"  100x version",
		RunE: func(_ *cobra.Command, _ []string) error {
			return renderVersion(f)
		},
	}
}

func renderVersion(f *factory.Factory) error {
	payload := versionPayload{
		Version:   version.Current.Version,
		Commit:    version.Current.Commit,
		BuildDate: version.Current.BuildDate,
	}
	return f.IO.Render(payload, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Version", Value: payload.Version},
			{Key: "Commit", Value: payload.Commit},
			{Key: "Build Date", Value: payload.BuildDate},
		})
	})
}

func configureUsageErrors(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return clierr.Usage(clierr.WithHelpHint(err, c.CommandPath()))
	})
	if cmd.Args != nil {
		args := cmd.Args
		cmd.Args = func(c *cobra.Command, a []string) error {
			if err := args(c, a); err != nil {
				return clierr.Usage(clierr.WithHelpHint(err, c.CommandPath()))
			}
			return nil
		}
	}
	// Cobra's built-in `required flag(s) ... not set` check runs after Args
	// and before RunE, and its error never flows through SetFlagErrorFunc.
	// Pre-empt it in PreRunE so we can attach the correct subcommand path
	// instead of the generic root help hint.
	prevPreRunE := cmd.PreRunE
	cmd.PreRunE = func(c *cobra.Command, a []string) error {
		if err := c.ValidateRequiredFlags(); err != nil {
			return clierr.Usage(clierr.WithHelpHint(err, c.CommandPath()))
		}
		if prevPreRunE != nil {
			return prevPreRunE(c, a)
		}
		return nil
	}
	if cmd.RunE != nil {
		runE := cmd.RunE
		cmd.RunE = func(c *cobra.Command, a []string) error {
			err := runE(c, a)
			if clierr.IsUsage(err) {
				return clierr.Usage(clierr.WithHelpHint(err, c.CommandPath()))
			}
			return err
		}
	}
	for _, sub := range cmd.Commands() {
		configureUsageErrors(sub)
	}
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
		return 0, clierr.Usagef("invalid --color %q (want auto, always, never)", s)
	}
}
