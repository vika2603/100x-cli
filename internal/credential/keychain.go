package credential

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const keychainService = "100x-cli"

type keychainStore struct{}

func (keychainStore) Save(profile, secret string) error {
	return keyring.Set(keychainService, profile, secret)
}

func (keychainStore) Load(profile string) (string, error) {
	v, err := keyring.Get(keychainService, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return v, err
}

func (keychainStore) Delete(profile string) error {
	err := keyring.Delete(keychainService, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
