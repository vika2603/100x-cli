package root

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

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

func TestSelectedPluralListShortcuts(t *testing.T) {
	for _, args := range [][]string{
		{"f", "orders", "--help"},
		{"f", "positions", "--help"},
		{"f", "triggers", "--help"},
		{"f", "balances", "--help"},
	} {
		stdout, stderr, err := executeRoot(t, args...)
		if err != nil {
			t.Fatalf("%q: %v", strings.Join(args, " "), err)
		}
		if stderr != "" {
			t.Fatalf("%q stderr=%q", strings.Join(args, " "), stderr)
		}
		if !strings.Contains(stdout, "Shortcut for `100x futures") {
			t.Fatalf("%q stdout=%q", strings.Join(args, " "), stdout)
		}
	}
}

func TestLongPluralShortcutsStayScoped(t *testing.T) {
	_, stderr, err := executeRoot(t, "orders")
	if err == nil {
		t.Fatal("expected top-level orders to stay unsupported")
	}
	if !strings.Contains(stderr, `unknown command "orders" for "100x"`) {
		t.Fatalf("stderr=%q", stderr)
	}

	for _, args := range [][]string{
		{"profiles"},
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

func TestUnknownCommandSuggestion(t *testing.T) {
	_, stderr, err := executeRoot(t, "futres")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "Did you mean this?") || !strings.Contains(stderr, "futures") {
		t.Fatalf("stderr=%q", stderr)
	}
}
