// Package config holds the CLI's profile model and TOML persistence layer.
//
// Secrets are deliberately not part of Profile: only ClientID is stored on
// disk; the matching ClientKey lives in the OS keychain (or chmod-600
// fallback) and is loaded via internal/credential.
package config

import (
	"fmt"
	"strings"
)

// DefaultEnv is the environment assumed by profile add when --env is omitted.
const DefaultEnv = "test"

var defaultEndpoints = map[string]string{
	"test": "https://api.lyantechinnovation.com/",
}

// Profile is one named credential binding.
//
// Secrets are NOT stored in the TOML file: only ClientID is persisted; the
// matching ClientKey lives in the OS keychain (or chmod 600 fallback file)
// keyed by the profile name.
type Profile struct {
	ClientID string `toml:"client_id"`
	// Env selects the endpoint from Config.Endpoints or the built-in defaults.
	Env string `toml:"env"`
}

// Config is the top-level TOML document.
type Config struct {
	Default   string             `toml:"default"`
	Endpoints map[string]string  `toml:"endpoints"`
	Profiles  map[string]Profile `toml:"profiles"`
}

// NormalizeEnv canonicalises env names used as endpoint-map keys.
func NormalizeEnv(env string) string {
	env = strings.TrimSpace(strings.ToLower(env))
	if env == "" {
		return DefaultEnv
	}
	return env
}

// EndpointForEnv resolves the URL for env from config overrides first, then
// built-in defaults.
func EndpointForEnv(c *Config, env string) (string, error) {
	env = NormalizeEnv(env)
	if c != nil && c.Endpoints != nil {
		if endpoint := strings.TrimSpace(c.Endpoints[env]); endpoint != "" {
			return endpoint, nil
		}
	}
	if endpoint := defaultEndpoints[env]; endpoint != "" {
		return endpoint, nil
	}
	return "", fmt.Errorf("no endpoint configured for env %q", env)
}

// EndpointForProfile resolves the endpoint selected by p.Env.
func EndpointForProfile(c *Config, p *Profile) (string, error) {
	if p == nil {
		return "", fmt.Errorf("profile is nil")
	}
	return EndpointForEnv(c, p.Env)
}

// SetEndpoint sets or overrides the global endpoint for env.
func SetEndpoint(c *Config, env, endpoint string) {
	if c.Endpoints == nil {
		c.Endpoints = map[string]string{}
	}
	c.Endpoints[NormalizeEnv(env)] = strings.TrimSpace(endpoint)
}
