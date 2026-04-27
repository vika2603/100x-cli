// Package config holds the CLI's profile model and TOML persistence layer.
//
// Secrets are deliberately not part of Profile: only ClientID is stored on
// disk; the matching ClientKey lives in the OS keychain (or chmod-600
// fallback) and is loaded via internal/credential.
package config

import (
	"fmt"
	"os"
	"strings"
)

// DefaultEndpoint is the built-in 100x API endpoint.
const DefaultEndpoint = "https://api.lyantechinnovation.com/"

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
// Precedence is: $E100X_ENDPOINT > built-in default.
func Endpoint() string {
	if endpoint := strings.TrimSpace(os.Getenv("E100X_ENDPOINT")); endpoint != "" {
		return endpoint
	}
	return DefaultEndpoint
}

// EndpointForProfile resolves the endpoint used with p.
func EndpointForProfile(p *Profile) (string, error) {
	if p == nil {
		return "", fmt.Errorf("profile is nil")
	}
	return Endpoint(), nil
}
