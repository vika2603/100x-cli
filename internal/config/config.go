// Package config holds the CLI's profile model and TOML persistence layer.
//
// Secrets are deliberately not part of Profile: only ClientID is stored on
// disk; the matching ClientKey lives in the OS keychain (or chmod-600
// fallback) and is loaded via internal/credential.
package config

// Profile is one named credential / endpoint binding.
//
// Secrets are NOT stored in the TOML file: only ClientID is persisted; the
// matching ClientKey lives in the OS keychain (or chmod 600 fallback file)
// keyed by the profile name.
type Profile struct {
	Endpoint string `toml:"endpoint"`
	ClientID string `toml:"client_id"`
	// Env is a free-text label such as "live" / "test" / "paper" used for the
	// confirmation prompt on destructive ops in the production environment.
	Env string `toml:"env"`
}

// Config is the top-level TOML document.
type Config struct {
	Default  string             `toml:"default"`
	Profiles map[string]Profile `toml:"profiles"`
}
