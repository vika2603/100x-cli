// Package config loads and resolves CLI profiles.
package config

import (
	"os"
	"path/filepath"
)

// AppName is the directory leaf used inside XDG paths.
const AppName = "100x"

// Dir returns the directory where config.toml lives.
//
// Honours $XDG_CONFIG_HOME, falling back to $HOME/.config.
func Dir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, AppName)
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".config", AppName)
	}
	return filepath.Join(".", "."+AppName)
}

// File is the path to the TOML profile file.
func File() string {
	return filepath.Join(Dir(), "config.toml")
}

// CredentialsDir is the per-profile secret-file fallback directory.
//
// Used when the OS keychain is unavailable. Files inside MUST be chmod 600.
func CredentialsDir() string {
	return filepath.Join(Dir(), "credentials")
}
