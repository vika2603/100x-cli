package config

import (
	"errors"
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
	// Endpoint is intentionally not part of Config: it is build-time injected
	// or supplied via $E100X_ENDPOINT, never persisted to disk.
	if strings.Contains(string(data), "endpoint") {
		t.Fatalf("config TOML must not encode an endpoint:\n%s", string(data))
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

// TestEndpointResolution covers precedence: $E100X_ENDPOINT > build-time
// DefaultEndpoint > ErrNoEndpoint. The build-time default is mutated via
// the package variable in tests, mirroring what -ldflags does at link time.
func TestEndpointResolution(t *testing.T) {
	saved := DefaultEndpoint
	t.Cleanup(func() { DefaultEndpoint = saved })

	t.Run("env beats build-time default", func(t *testing.T) {
		DefaultEndpoint = "https://build.example.com"
		t.Setenv("E100X_ENDPOINT", "https://env.example.com")
		got, err := Endpoint()
		if err != nil || got != "https://env.example.com" {
			t.Errorf("Endpoint=%q err=%v want https://env.example.com", got, err)
		}
	})

	t.Run("build-time default used when env empty", func(t *testing.T) {
		DefaultEndpoint = "https://build.example.com"
		t.Setenv("E100X_ENDPOINT", "")
		got, err := Endpoint()
		if err != nil || got != "https://build.example.com" {
			t.Errorf("Endpoint=%q err=%v want https://build.example.com", got, err)
		}
	})

	t.Run("nothing configured returns ErrNoEndpoint", func(t *testing.T) {
		DefaultEndpoint = ""
		t.Setenv("E100X_ENDPOINT", "")
		_, err := Endpoint()
		if !errors.Is(err, ErrNoEndpoint) {
			t.Errorf("err=%v want ErrNoEndpoint", err)
		}
	})

	t.Run("whitespace-only values do not satisfy", func(t *testing.T) {
		DefaultEndpoint = "   "
		t.Setenv("E100X_ENDPOINT", "   ")
		_, err := Endpoint()
		if !errors.Is(err, ErrNoEndpoint) {
			t.Errorf("err=%v want ErrNoEndpoint", err)
		}
	})

	t.Run("rejects garbage env value", func(t *testing.T) {
		DefaultEndpoint = ""
		t.Setenv("E100X_ENDPOINT", "not-a-url")
		_, err := Endpoint()
		if err == nil || !strings.Contains(err.Error(), "invalid endpoint") {
			t.Errorf("err=%v want validation error", err)
		}
	})

	t.Run("rejects non-http scheme", func(t *testing.T) {
		DefaultEndpoint = ""
		t.Setenv("E100X_ENDPOINT", "ftp://example.com")
		_, err := Endpoint()
		if err == nil || !strings.Contains(err.Error(), "scheme") {
			t.Errorf("err=%v want scheme error", err)
		}
	})

	t.Run("rejects missing host", func(t *testing.T) {
		DefaultEndpoint = ""
		t.Setenv("E100X_ENDPOINT", "https://")
		_, err := Endpoint()
		if err == nil || !strings.Contains(err.Error(), "host") {
			t.Errorf("err=%v want host error", err)
		}
	})

	t.Run("rejects garbage build-time default", func(t *testing.T) {
		DefaultEndpoint = "not-a-url"
		t.Setenv("E100X_ENDPOINT", "")
		_, err := Endpoint()
		if err == nil || !strings.Contains(err.Error(), "invalid endpoint") {
			t.Errorf("err=%v want validation error", err)
		}
	})
}
