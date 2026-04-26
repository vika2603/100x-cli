package credential

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vika2603/100x-cli/internal/config"
)

type fileStore struct{}

func (fileStore) path(profile string) string {
	return filepath.Join(config.CredentialsDir(), filepath.Base(profile))
}

func (s fileStore) Save(profile, secret string) error {
	if err := os.MkdirAll(config.CredentialsDir(), 0o700); err != nil {
		return fmt.Errorf("mkdir credentials: %w", err)
	}
	p := s.path(profile)
	return os.WriteFile(p, []byte(secret), 0o600)
}

func (s fileStore) Load(profile string) (string, error) {
	p := s.path(profile)
	data, err := os.ReadFile(p) // #nosec G304 -- p is XDG-derived plus filepath.Base(profile), not arbitrary input
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

func (s fileStore) Delete(profile string) error {
	p := s.path(profile)
	err := os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
