// Package root assembles the top-level cobra command tree.
package root

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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
		Short: "100x futures-trading CLI",
		Long: "Use 100x from the terminal for market data, balances, orders, triggers, and positions.\n\n" +
			"Private commands read credentials from a named profile. A profile stores user identity\n" +
			"and env selection; endpoint settings live under [env.<name>] in config. Public market\n" +
			"commands can run without private credentials as long as an endpoint is configured.\n\n" +
			"Human output is designed for terminal use. Add --json for machine-readable output, and\n" +
			"use --jq to filter that JSON when scripting. Use --help on any subcommand to inspect\n" +
			"required arguments, default values, examples, and command-specific notes.",
		Example: "# Add a test profile named test, using env test and storing the secret in the keychain\n" +
			"  100x profile add test --env test --client-id <CID>\n\n" +
			"# Show the latest ticker-style state for BTCUSDT\n" +
			"  100x futures market state BTCUSDT\n\n" +
			"# Place a BUY limit order on BTCUSDT at 70000 for size 0.001\n" +
			"  100x futures order place BTCUSDT --side buy --price 70000 --size 0.001",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands"},
		&cobra.Group{ID: "auth", Title: "Auth Commands"},
		&cobra.Group{ID: "tools", Title: "Tooling"},
	)

	cmd.PersistentFlags().StringVar(&gf.profile, "profile", "", "use credentials from profile <name>")
	cmd.PersistentFlags().BoolVar(&gf.jsonOut, "json", false, "write JSON to stdout")
	cmd.PersistentFlags().StringVar(&gf.jq, "jq", "", "run a gojq expression against JSON output")
	cmd.PersistentFlags().BoolVarP(&gf.quiet, "quiet", "q", false, "hide human-readable stdout")
	cmd.PersistentFlags().BoolVarP(&gf.yes, "yes", "y", false, "answer yes to confirmation prompts")
	cmd.PersistentFlags().StringVar(&gf.color, "color", "auto", "color mode: auto | always | never (NO_COLOR honored)")
	cmd.PersistentFlags().DurationVar(&gf.timeout, "timeout", 30*time.Second, "HTTP timeout per request")
	_ = cmd.RegisterFlagCompletionFunc("color", cobra.FixedCompletions([]string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp))

	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		if isCredentialFreeCmd(c) {
			r, err := newRenderer(gf)
			if err != nil {
				return err
			}
			f.IO = r
			f.Yes = gf.yes
			return nil
		}
		if isPublicCmd(c) {
			return populatePublic(f, gf)
		}
		return populate(f, gf)
	}

	futuresCmd := futuresGroup.NewCmdFutures(f)
	futuresCmd.GroupID = "core"
	profileCmd := profile.NewCmdProfile(f)
	profileCmd.GroupID = "auth"
	completionCmd := completion.NewCmdCompletion()
	completionCmd.GroupID = "tools"
	versionCmd := newCmdVersion(f)
	versionCmd.GroupID = "tools"
	cmd.AddCommand(futuresCmd, profileCmd, completionCmd, versionCmd)
	configureHelp(cmd)

	emit := func(err error, _ int, codeString string) {
		if gf.jsonOut || gf.jq != "" {
			payload := struct {
				Error errorPayload `json:"error"`
			}{Error: errorPayload{Code: codeString, Message: err.Error()}}
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
		fmt.Fprintln(os.Stderr, "error:", humanErrorMessage(err, codeString))
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

// isCredentialFreeCmd reports whether c is in a subtree that does not need
// a Factory.Client. profile add / list / use / show / remove and completion
// run before any credentials exist.
func isCredentialFreeCmd(c *cobra.Command) bool {
	for cur := c; cur != nil; cur = cur.Parent() {
		switch cur.Name() {
		case "profile", "completion", "help", "version":
			return true
		}
	}
	return false
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
			payload := versionPayload{
				Version:   version.Version,
				Commit:    version.Commit,
				BuildDate: version.BuildDate,
			}
			return f.IO.Render(payload, func() error {
				return f.IO.Object([]output.KV{
					{Key: "Version", Value: payload.Version},
					{Key: "Commit", Value: payload.Commit},
					{Key: "Build Date", Value: payload.BuildDate},
				})
			})
		},
	}
}

// isPublicCmd reports whether c is in a subtree that can use a configured
// endpoint without loading private credentials.
func isPublicCmd(c *cobra.Command) bool {
	for cur := c; cur != nil; cur = cur.Parent() {
		if cur.Name() == "market" {
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
	endpoint, err := config.EndpointForProfile(cfg, p)
	if err != nil {
		return fmt.Errorf("resolve endpoint for profile %q: %w", name, err)
	}
	f.ProfileName = name
	f.Profile = p
	f.Client = futures.New(futures.Options{
		Endpoint:   endpoint,
		ClientID:   p.ClientID,
		ClientKey:  secret,
		HTTPClient: &http.Client{Timeout: gf.timeout},
	})
	return nil
}

func populatePublic(f *factory.Factory, gf *globalFlags) error {
	r, err := newRenderer(gf)
	if err != nil {
		return err
	}
	f.IO = r
	f.Yes = gf.yes
	f.Timeout = gf.timeout

	if os.Getenv("E100X_FAKE") == "1" {
		f.Client = futures.NewWithDoer(fakeapi.New())
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name, p, err := config.Resolve(cfg, gf.profile)
	if err != nil {
		if errors.Is(err, config.ErrNoProfile) {
			return fmt.Errorf("no profile configured: run `100x profile add`")
		}
		return err
	}
	endpoint, err := config.EndpointForProfile(cfg, p)
	if err != nil {
		return fmt.Errorf("resolve endpoint for profile %q: %w", name, err)
	}
	f.ProfileName = name
	f.Profile = p
	f.Client = futures.New(futures.Options{
		Endpoint:   endpoint,
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
