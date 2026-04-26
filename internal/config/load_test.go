package config

import (
	"os"
	"path/filepath"
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
			"live": {Endpoint: "https://api.example.com", ClientID: "abc", Env: "live"},
			"test": {Endpoint: "https://test.example.com", ClientID: "xyz", Env: "test"},
		},
	}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}

	// File mode must be 0600.
	info, err := os.Stat(filepath.Join(dir, AppName, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v want 0600", info.Mode().Perm())
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "live" {
		t.Errorf("Default=%q want live", got.Default)
	}
	if got.Profiles["live"].Endpoint != "https://api.example.com" {
		t.Errorf("Profiles[live].Endpoint=%q want https://api.example.com", got.Profiles["live"].Endpoint)
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
			"d": {Endpoint: "default"},
			"e": {Endpoint: "env"},
			"x": {Endpoint: "explicit"},
		},
	}

	t.Run("default", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "")
		name, p, err := Resolve(c, "")
		if err != nil || name != "d" || p.Endpoint != "default" {
			t.Errorf("got %s/%v err=%v", name, p, err)
		}
	})
	t.Run("env override", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "e")
		name, p, _ := Resolve(c, "")
		if name != "e" || p.Endpoint != "env" {
			t.Errorf("got %s/%v", name, p)
		}
	})
	t.Run("explicit beats env", func(t *testing.T) {
		t.Setenv("E100X_PROFILE", "e")
		name, p, _ := Resolve(c, "x")
		if name != "x" || p.Endpoint != "explicit" {
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
