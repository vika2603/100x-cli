package credential

import (
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

type deleteStore struct {
	err error
}

func (s deleteStore) Save(string, []byte) error { return nil }
func (s deleteStore) Load(string) ([]byte, error) {
	return nil, ErrNotFound
}
func (s deleteStore) Delete(string) error { return s.err }

func TestChainDeleteReturnsFailureWhenNoBackendDeletes(t *testing.T) {
	want := errors.New("delete failed")
	store := &chain{stores: []Store{deleteStore{err: want}}}

	if err := store.Delete("id"); !errors.Is(err, want) {
		t.Fatalf("err=%v want %v", err, want)
	}
}

func TestChainDeleteIgnoresUnavailableBackendAfterSuccess(t *testing.T) {
	store := &chain{stores: []Store{
		deleteStore{err: errors.New("keychain unavailable")},
		deleteStore{},
	}}

	if err := store.Delete("id"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestChainDeleteReturnsFailureWhenOnlyMissingBackendsSucceed(t *testing.T) {
	want := errors.New("delete failed")
	store := &chain{stores: []Store{
		deleteStore{err: ErrNotFound},
		deleteStore{err: want},
	}}

	if err := store.Delete("id"); !errors.Is(err, want) {
		t.Fatalf("err=%v want %v", err, want)
	}
}

func TestChainDeleteMissingEverywhereIsNoop(t *testing.T) {
	store := &chain{stores: []Store{
		deleteStore{err: ErrNotFound},
		deleteStore{err: ErrNotFound},
	}}

	if err := store.Delete("id"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestFileStorePathUsesFullClientID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := fileStore{}

	if s.path("team/a") == s.path("a") {
		t.Fatal("fileStore path should not collapse client IDs with the same base name")
	}
}

// TestSaveSecretRejectsEmptyClientKey: refuse to persist an envelope that
// would silently sign with an empty key. Better to fail at the boundary
// than ship a credential nobody can authenticate with.
func TestSaveSecretRejectsEmptyClientKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	keyring.MockInit()
	err := SaveSecret("id", Envelope{ClientID: "id"})
	if err == nil || !strings.Contains(err.Error(), "client_key") {
		t.Fatalf("err=%v want client_key validation", err)
	}
}

// TestLoadSecretRejectsEnvelopeMissingClientKey: an envelope that
// deserializes with an empty ClientKey must surface as a local error
// instead of silently signing with an empty key.
func TestLoadSecretRejectsEnvelopeMissingClientKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	keyring.MockInit()
	if err := keyring.Set(keychainService, "id", `{"client_id":"id"}`); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSecret("id")
	if err == nil || !strings.Contains(err.Error(), "client_key") {
		t.Fatalf("err=%v want client_key validation", err)
	}
}
