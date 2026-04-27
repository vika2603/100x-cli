package root

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/exit"
)

func executeRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	stdout, stderr, restore := captureStdoutStderr(t)

	cmd, emit := NewCmdRoot()
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		_, code := exit.Classify(err)
		emit(err, 0, code)
	}
	restore()
	return stdout.String(), stderr.String(), err
}

func captureStdoutStderr(t *testing.T) (*bytes.Buffer, *bytes.Buffer, func()) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return stdout, stderr, func() {
		_ = outW.Close()
		_ = errW.Close()
		os.Stdout, os.Stderr = origOut, origErr
		_, _ = io.Copy(stdout, outR)
		_, _ = io.Copy(stderr, errR)
		_ = outR.Close()
		_ = errR.Close()
	}
}

func TestVersionCommandJSON(t *testing.T) {
	stdout, stderr, err := executeRoot(t, "--json", "version")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q", stderr)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid json %q: %v", stdout, err)
	}
	if got["version"] == "" || got["commit"] == "" || got["build_date"] == "" {
		t.Fatalf("missing version fields: %#v", got)
	}
}

func TestHelpListsCommandAliases(t *testing.T) {
	stdout, stderr, err := executeRoot(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q", stderr)
	}
	if !strings.Contains(stdout, "futures (f)") {
		t.Fatalf("root help missing futures alias: %q", stdout)
	}
	if !strings.Contains(stdout, "profile (prof)") {
		t.Fatalf("root help missing profile aliases: %q", stdout)
	}

	stdout, stderr, err = executeRoot(t, "f", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q", stderr)
	}
	if !strings.Contains(stdout, "order (o)") {
		t.Fatalf("futures help missing order alias: %q", stdout)
	}
	if !strings.Contains(stdout, "balance (bal)") {
		t.Fatalf("futures help missing balance alias: %q", stdout)
	}
}

func TestVersionFlagIsNotSupported(t *testing.T) {
	_, stderr, err := executeRoot(t, "--version")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "unknown flag: --version") {
		t.Fatalf("stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "Run `100x --help` for usage") {
		t.Fatalf("stderr missing help hint: %q", stderr)
	}
}

func TestPluralVerbsAreNotAccepted(t *testing.T) {
	for _, args := range [][]string{
		{"orders"},
		{"profiles"},
		{"f", "orders"},
		{"f", "positions"},
		{"f", "triggers"},
		{"f", "balances"},
		{"f", "markets"},
	} {
		_, _, err := executeRoot(t, args...)
		if err == nil {
			t.Fatalf("expected %q to be rejected", strings.Join(args, " "))
		}
	}
}

func TestVerboseSynonymAliasesAreNotAccepted(t *testing.T) {
	cases := [][]string{
		{"profile", "create"},
		{"profile", "default"},
		{"profile", "switch"},
		{"f", "market", "ticker"},
		{"f", "market", "candles"},
		{"f", "order", "trades"},
	}
	for _, args := range cases {
		_, _, err := executeRoot(t, args...)
		if err == nil {
			t.Fatalf("expected %q to be rejected", strings.Join(args, " "))
		}
	}
}

// TestAuthAnnotationsLockBoundary asserts the per-command auth declarations
// the rest of the codebase depends on. It walks the command tree and checks
// that each verb resolves to the AuthMode required by its semantics: market
// children → public, other futures verbs → private, profile/version/
// completion → none. Adding a new public verb that forgets to override the
// default, or accidentally promoting a market verb to private, fails here.
func TestAuthAnnotationsLockBoundary(t *testing.T) {
	cmd, _ := NewCmdRoot()

	cases := []struct {
		path []string
		want factory.AuthMode
	}{
		{[]string{"version"}, factory.AuthNone},
		{[]string{"completion", "bash"}, factory.AuthNone},
		{[]string{"profile", "list"}, factory.AuthNone},
		{[]string{"profile", "add"}, factory.AuthNone},

		{[]string{"futures", "market", "list"}, factory.AuthPublic},
		{[]string{"futures", "market", "state"}, factory.AuthPublic},
		{[]string{"futures", "market", "depth"}, factory.AuthPublic},
		{[]string{"futures", "market", "kline"}, factory.AuthPublic},
		{[]string{"futures", "market", "deals"}, factory.AuthPublic},

		{[]string{"futures", "balance", "list"}, factory.AuthPrivate},
		{[]string{"futures", "balance", "history"}, factory.AuthPrivate},
		{[]string{"futures", "order", "list"}, factory.AuthPrivate},
		{[]string{"futures", "order", "place"}, factory.AuthPrivate},
		{[]string{"futures", "order", "edit"}, factory.AuthPrivate},
		{[]string{"futures", "trigger", "place"}, factory.AuthPrivate},
		{[]string{"futures", "trigger", "edit"}, factory.AuthPrivate},
		{[]string{"futures", "position", "list"}, factory.AuthPrivate},
		{[]string{"futures", "position", "close"}, factory.AuthPrivate},
		{[]string{"futures", "preference"}, factory.AuthPrivate},
	}

	for _, tc := range cases {
		c, _, err := cmd.Find(tc.path)
		if err != nil {
			t.Errorf("Find(%v): %v", tc.path, err)
			continue
		}
		if got := factory.LookupAuth(c); got != tc.want {
			t.Errorf("LookupAuth(%v)=%q want %q", tc.path, got, tc.want)
		}
	}
}

// TestGroupCommandsAreSkippedStructurally documents the structural rule that
// complements the annotation system: group commands (those with subcommands)
// always skip session.Load, regardless of any inherited annotation, because
// their RunE only calls c.Help().
func TestGroupCommandsAreSkippedStructurally(t *testing.T) {
	cmd, _ := NewCmdRoot()
	groups := [][]string{
		{},                      // root
		{"futures"},             // futures group
		{"futures", "market"},   // public group
		{"futures", "order"},    // private group
		{"futures", "trigger"},  // private group
		{"futures", "position"}, // private group
		{"futures", "balance"},  // private group
		{"profile"},             // auth group
	}
	for _, path := range groups {
		c, _, err := cmd.Find(path)
		if err != nil {
			t.Errorf("Find(%v): %v", path, err)
			continue
		}
		if !c.HasSubCommands() {
			t.Errorf("%v: expected HasSubCommands()=true (otherwise PreRunE would try to load a client for it)", path)
		}
	}
	// Sanity: a leaf verb really does report HasSubCommands=false so the
	// structural skip would not over-apply.
	leaf, _, err := cmd.Find([]string{"futures", "balance", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if leaf.HasSubCommands() {
		t.Fatalf("balance list must be a leaf, got HasSubCommands=true")
	}
}

// TestRequireAnnotationOverridesParent verifies the inheritance rule: a
// child's explicit declaration takes precedence over the parent's. The
// market group sits under futures (RequirePrivate) and overrides with
// RequirePublic; every market descendant must resolve to AuthPublic.
func TestRequireAnnotationOverridesParent(t *testing.T) {
	cmd, _ := NewCmdRoot()
	market, _, err := cmd.Find([]string{"futures", "market"})
	if err != nil {
		t.Fatal(err)
	}
	if got := factory.LookupAuth(market); got != factory.AuthPublic {
		t.Fatalf("market group LookupAuth=%q want AuthPublic (must override the futures default)", got)
	}
	// Walk every market child and confirm they all inherit AuthPublic.
	for _, child := range market.Commands() {
		if got := factory.LookupAuth(child); got != factory.AuthPublic {
			t.Errorf("market child %q inherits %q want AuthPublic", child.Name(), got)
		}
	}
}

// TestAuthLookupFallsThroughCleanly is a regression guard: an unmarked
// command outside any annotated subtree must return AuthNone, not panic.
func TestAuthLookupFallsThroughCleanly(t *testing.T) {
	c := &cobra.Command{Use: "x"}
	if got := factory.LookupAuth(c); got != factory.AuthNone {
		t.Errorf("LookupAuth(unmarked)=%q want AuthNone", got)
	}
}

func TestUnknownCommandSuggestion(t *testing.T) {
	_, stderr, err := executeRoot(t, "futres")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "Did you mean this?") || !strings.Contains(stderr, "futures") {
		t.Fatalf("stderr=%q", stderr)
	}
}

// TestRequiredFlagPointsToSubcommand checks the help-hint regression: cobra's
// `required flag(s) ... not set` error must direct the user to the
// subcommand's --help, not the root --help. The command path under test is
// AuthPrivate, so install a fake profile + keychain credential up front so
// the persistent-prerun gate does not pre-empt the required-flag check.
func TestRequiredFlagPointsToSubcommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("E100X_ENDPOINT", "https://test.invalid")
	t.Setenv("E100X_PROFILE", "")
	keyring.MockInit()
	if err := config.Save(&config.Config{
		Default:  "test",
		Profiles: map[string]config.Profile{"test": {ClientID: "id-test"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-test", credential.Envelope{
		ClientID: "id-test", ClientKey: "secret-test",
	}); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := executeRoot(t, "futures", "order", "place", "BTCUSDT")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "required flag(s)") {
		t.Fatalf("stderr=%q want required-flag error", stderr)
	}
	if !strings.Contains(stderr, "100x futures order place --help") {
		t.Fatalf("stderr=%q want subcommand help hint", stderr)
	}
	if strings.Contains(stderr, "Run `100x --help`") {
		t.Fatalf("stderr=%q must not point to root help", stderr)
	}
}
