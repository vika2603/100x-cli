// Package credential stores per-profile API secrets, preferring the OS
// keychain and falling back to a chmod-600 file.
package credential

import (
	"errors"
)

// ErrNotFound is returned when the requested credential is missing from every backend.
var ErrNotFound = errors.New("credential not found")

// Store reads and writes profile credentials.
type Store interface {
	// Save persists the secret for `profile`.
	Save(profile, secret string) error
	// Load returns the secret for `profile` or ErrNotFound.
	Load(profile string) (string, error)
	// Delete removes the secret for `profile`. Missing values are a no-op.
	Delete(profile string) error
}

// Default returns the platform-preferred Store.
//
// Order: OS keychain (zalando/go-keyring) → chmod 600 file in CredentialsDir.
func Default() Store {
	return &chain{
		stores: []Store{
			&keychainStore{},
			&fileStore{},
		},
	}
}

type chain struct {
	stores []Store
}

func (c *chain) Save(profile, secret string) error {
	var lastErr error
	for _, s := range c.stores {
		if err := s.Save(profile, secret); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		return errors.New("no credential store available")
	}
	return lastErr
}

func (c *chain) Load(profile string) (string, error) {
	for _, s := range c.stores {
		v, err := s.Load(profile)
		if err == nil {
			return v, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return "", err
		}
	}
	return "", ErrNotFound
}

func (c *chain) Delete(profile string) error {
	for _, s := range c.stores {
		_ = s.Delete(profile)
	}
	return nil
}
