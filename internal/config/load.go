package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ErrNoProfile is returned when no profile is configured.
var ErrNoProfile = errors.New("no profile configured")

// Load reads the TOML config file. Returns an empty Config (no error) when
// the file is absent — first-run is not a failure.
func Load() (*Config, error) {
	path := File()
	data, err := os.ReadFile(path) // #nosec G304 -- path is derived from XDG config dir, not user input
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	return &c, nil
}

// Save writes the config back to disk, creating the directory if needed.
// Files are written with 0600 mode; directories with 0700.
func Save(c *Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", Dir(), err)
	}
	path := File()
	tmp, err := os.CreateTemp(Dir(), ".config.toml.")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(c); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode toml: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Resolve picks the named profile, or the default when name is empty.
//
// Precedence is: explicit name > $E100X_PROFILE > Config.Default.
func Resolve(c *Config, requested string) (string, *Profile, error) {
	name := requested
	if name == "" {
		name = os.Getenv("E100X_PROFILE")
	}
	if name == "" {
		name = c.Default
	}
	if name == "" {
		return "", nil, ErrNoProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return "", nil, fmt.Errorf("profile %q not found", name)
	}
	return name, &p, nil
}
