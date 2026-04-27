package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveLoadRoundTrip writes a Config and reads it back from a temp XDG
// directory, exercising both directory creation and TOML encoding.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	c := &Config{
		Default: "live",
		Profiles: map[string]Profile{
			"live": {ClientID: "abc"},
			"test": {ClientID: "xyz"},
		},
	}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, AppName, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(filepath.Join(dir, AppName, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "endpoint") || strings.Contains(string(data), "[env.") {
		t.Fatalf("config TOML must not encode endpoint/env settings:\n%s", string(data))
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "live" {
		t.Errorf("Default=%q want live", got.Default)
	}
	if len(got.Profiles) != 2 {
		t.Errorf("len(Profiles)=%d want 2", len(got.Profiles))
	}
}

// TestLoadMissingFile returns an empty config and no error so first-run is
// not a failure mode.
func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "" {
		t.Errorf("Default=%q want empty", got.Default)
	}
	if got.Profiles == nil {
		t.Error("Profiles must be non-nil")
	}
}

// TestResolve covers precedence: explicit > env > default.
func TestResolve(t *testing.T) {
	c := &Config{
		Default: "d",
		Profiles: map[string]Profile{
			"d": {ClientID: "default"},
			"e": {ClientID: "env"},
			"x": {ClientID: "explicit"},
		},
	}

	t.Run("default", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "")
		name, p, err := Resolve(c, "")
		if err != nil || name != "d" || p.ClientID != "default" {
			t.Errorf("got %s/%v err=%v", name, p, err)
		}
	})
	t.Run("env override", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "e")
		name, p, _ := Resolve(c, "")
		if name != "e" || p.ClientID != "env" {
			t.Errorf("got %s/%v", name, p)
		}
	})
	t.Run("explicit beats env", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "e")
		name, p, _ := Resolve(c, "x")
		if name != "x" || p.ClientID != "explicit" {
			t.Errorf("got %s/%v", name, p)
		}
	})
	t.Run("missing profile", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "")
		_, _, err := Resolve(c, "nope")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestEndpointResolution(t *testing.T) {
	t.Setenv("E100X_ENDPOINT", "")
	if got := Endpoint(); got != DefaultEndpoint {
		t.Errorf("Endpoint(default)=%q want %q", got, DefaultEndpoint)
	}

	t.Setenv("E100X_ENDPOINT", "https://env.example.com")
	if got := Endpoint(); got != "https://env.example.com" {
		t.Errorf("Endpoint(env)=%q want https://env.example.com", got)
	}
}
