// Package credential stores API secrets keyed by client_id, preferring the
// OS keychain and falling back to a chmod-600 file.
//
// client_id is the API identity — two profiles that point at the same
// client_id naturally share one secret. Every entry is a JSON Envelope so
// future fields (key type, rotation timestamps, etc.) can be added without
// rewriting the storage layer.
package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when the requested credential is missing from every backend.
var ErrNotFound = errors.New("credential not found")

// Envelope is the on-disk shape of one stored credential.
type Envelope struct {
	ClientID  string    `json:"client_id"`
	ClientKey string    `json:"client_key"`
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Store reads and writes opaque blobs keyed by client_id. Backends do not
// know or care about the JSON schema inside.
type Store interface {
	Save(clientID string, blob []byte) error
	Load(clientID string) ([]byte, error)
	Delete(clientID string) error
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

// SaveSecret encodes env as JSON and writes it under clientID via the default Store.
func SaveSecret(clientID string, env Envelope) error {
	if clientID == "" {
		return errors.New("save credential: empty client_id")
	}
	if env.ClientKey == "" {
		return errors.New("save credential: empty client_key")
	}
	if env.ClientID == "" {
		env.ClientID = clientID
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now().UTC()
	}
	blob, err := json.Marshal(env) // #nosec G117 -- marshaling the credential envelope IS the storage path
	if err != nil {
		return fmt.Errorf("encode credential envelope: %w", err)
	}
	return Default().Save(clientID, blob)
}

// LoadSecret reads and decodes the Envelope stored under clientID. Returns
// ErrNotFound when no backend has the entry. Rejects envelopes whose
// client_key is empty so a malformed entry surfaces locally instead of
// reaching the API as an unsigned request.
func LoadSecret(clientID string) (Envelope, error) {
	if clientID == "" {
		return Envelope{}, ErrNotFound
	}
	blob, err := Default().Load(clientID)
	if err != nil {
		return Envelope{}, err
	}
	var env Envelope
	if err := json.Unmarshal(blob, &env); err != nil {
		return Envelope{}, fmt.Errorf("decode credential envelope: %w", err)
	}
	if env.ClientKey == "" {
		return Envelope{}, fmt.Errorf("credential envelope for %q is missing client_key; re-run `100x profile add`", clientID)
	}
	return env, nil
}

// DeleteSecret removes the entry under clientID. Missing entries are a no-op.
func DeleteSecret(clientID string) error {
	if clientID == "" {
		return nil
	}
	return Default().Delete(clientID)
}

type chain struct {
	stores []Store
}

func (c *chain) Save(clientID string, blob []byte) error {
	var lastErr error
	for _, s := range c.stores {
		if err := s.Save(clientID, blob); err != nil {
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

func (c *chain) Load(clientID string) ([]byte, error) {
	for _, s := range c.stores {
		v, err := s.Load(clientID)
		if err == nil {
			return v, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}

func (c *chain) Delete(clientID string) error {
	var lastErr error
	deleted := false
	for _, s := range c.stores {
		if err := s.Delete(clientID); err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			lastErr = err
			continue
		}
		deleted = true
	}
	if !deleted && lastErr != nil {
		return lastErr
	}
	return nil
}
