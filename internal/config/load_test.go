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
		Endpoints: map[string]string{
			"live": "https://api.example.com",
			"test": "https://test.example.com",
		},
		Profiles: map[string]Profile{
			"live": {ClientID: "abc", Env: "live"},
			"test": {ClientID: "xyz", Env: "test"},
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
	if got.Endpoints["live"] != "https://api.example.com" {
		t.Errorf("Endpoints[live]=%q want https://api.example.com", got.Endpoints["live"])
	}
	if len(got.Profiles) != 2 {
		t.Errorf("len(Profiles)=%d want 2", len(got.Profiles))
	}
	profile := got.Profiles["test"]
	endpoint, err := EndpointForProfile(got, &profile)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "https://test.example.com" {
		t.Errorf("EndpointForProfile(test)=%q want https://test.example.com", endpoint)
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
	if got.Endpoints == nil {
		t.Error("Endpoints must be non-nil")
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
	c := &Config{
		Endpoints: map[string]string{
			"live": "https://live.example.com",
			"test": "https://override.example.com",
		},
	}

	got, err := EndpointForEnv(c, " live ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://live.example.com" {
		t.Errorf("EndpointForEnv(live)=%q want https://live.example.com", got)
	}

	got, err = EndpointForEnv(c, "test")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://override.example.com" {
		t.Errorf("EndpointForEnv(test)=%q want https://override.example.com", got)
	}

	got, err = EndpointForEnv(&Config{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.lyantechinnovation.com/" {
		t.Errorf("EndpointForEnv(empty)=%q want built-in test endpoint", got)
	}

	if _, err := EndpointForEnv(&Config{}, "live"); err == nil {
		t.Fatal("expected missing live endpoint error")
	}
}
