// Package config holds the CLI's profile model and TOML persistence layer.
//
// Secrets are deliberately not part of Profile: only ClientID is stored on
// disk; the matching ClientKey lives in the OS keychain (or chmod-600
// fallback) and is loaded via internal/credential.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// DefaultEndpoint is the build-time API base URL. It is injected via
// -ldflags at link time (see Makefile / .goreleaser.yaml). Leaving it
// empty in `go run` / unflagged `go build` forces the developer to set
// $E100X_ENDPOINT so we never silently hit the wrong host.
var DefaultEndpoint = ""

// ErrNoEndpoint is returned when neither $E100X_ENDPOINT nor the
// build-time DefaultEndpoint is set.
var ErrNoEndpoint = errors.New("no API endpoint configured")

// Profile is one named credential binding.
//
// Secrets are NOT stored in the TOML file: only ClientID is persisted; the
// matching ClientKey lives in the OS keychain (or chmod 600 fallback file)
// keyed by the profile name.
type Profile struct {
	ClientID string `toml:"client_id"`
}

// Config is the top-level TOML document.
type Config struct {
	Default  string             `toml:"default"`
	Profiles map[string]Profile `toml:"profiles"`
}

// Endpoint resolves the API base URL.
//
// Precedence is: $E100X_ENDPOINT > build-time DefaultEndpoint. Returns
// ErrNoEndpoint when neither is set. The winning value is validated as an
// absolute http(s) URL with a host; an invalid value is rejected up front
// instead of failing on the first request.
func Endpoint() (string, error) {
	if endpoint := strings.TrimSpace(os.Getenv("E100X_ENDPOINT")); endpoint != "" {
		return validateEndpoint(endpoint)
	}
	if endpoint := strings.TrimSpace(DefaultEndpoint); endpoint != "" {
		return validateEndpoint(endpoint)
	}
	return "", ErrNoEndpoint
}

func validateEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid endpoint %q: scheme must be http or https", raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid endpoint %q: missing host", raw)
	}
	return raw, nil
}
