package credential

import (
	"errors"
	"testing"
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
