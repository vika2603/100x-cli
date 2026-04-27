package credential

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vika2603/100x-cli/internal/config"
)

const fileExt = ".json"

type fileStore struct{}

func (fileStore) path(clientID string) string {
	sum := sha256.Sum256([]byte(clientID))
	return filepath.Join(config.CredentialsDir(), hex.EncodeToString(sum[:])+fileExt)
}

func (s fileStore) Save(clientID string, blob []byte) error {
	if err := os.MkdirAll(config.CredentialsDir(), 0o700); err != nil {
		return fmt.Errorf("mkdir credentials: %w", err)
	}
	return os.WriteFile(s.path(clientID), blob, 0o600)
}

func (s fileStore) Load(clientID string) ([]byte, error) {
	p := s.path(clientID)
	data, err := os.ReadFile(p) // #nosec G304 -- p is XDG-derived plus filepath.Base(clientID), not arbitrary input
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s fileStore) Delete(clientID string) error {
	err := os.Remove(s.path(clientID))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}
