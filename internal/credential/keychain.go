package credential

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const keychainService = "100x-cli"

type keychainStore struct{}

func (keychainStore) Save(clientID string, blob []byte) error {
	return keyring.Set(keychainService, clientID, string(blob))
}

func (keychainStore) Load(clientID string) ([]byte, error) {
	v, err := keyring.Get(keychainService, clientID)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return []byte(v), nil
}

func (keychainStore) Delete(clientID string) error {
	err := keyring.Delete(keychainService, clientID)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
