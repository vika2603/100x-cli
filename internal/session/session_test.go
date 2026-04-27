package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
)

// isolate redirects XDG_CONFIG_HOME, E100X_ENDPOINT, E100X_PROFILE to test
// values and swaps the keyring for an in-memory mock so a real keychain is
// never touched.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("E100X_ENDPOINT", "")
	t.Setenv("E100X_PROFILE", "")
	keyring.MockInit()
}

// TestLoadPublicSkipsCredentials verifies that Public:true builds a client
// from config.Endpoint() without touching profile or credential storage.
func TestLoadPublicSkipsCredentials(t *testing.T) {
	isolate(t)
	t.Setenv("E100X_ENDPOINT", "https://public.example.com")

	sess, err := Load(LoadOptions{Public: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("Load(Public): %v", err)
	}
	if sess.Client == nil {
		t.Fatal("Client must not be nil")
	}
	if sess.Profile != nil || sess.ProfileName != "" {
		t.Errorf("Public session leaked profile: %+v / %q", sess.Profile, sess.ProfileName)
	}
	if sess.Endpoint != "https://public.example.com" {
		t.Errorf("Endpoint=%q want https://public.example.com", sess.Endpoint)
	}
}

// TestLoadPrivateNoProfileReturnsErrNoProfile keeps callers in charge of
// rendering the "first run" message: Load surfaces config.ErrNoProfile
// untouched when nothing is configured.
func TestLoadPrivateNoProfileReturnsErrNoProfile(t *testing.T) {
	isolate(t)

	_, err := Load(LoadOptions{Timeout: time.Second})
	if !errors.Is(err, config.ErrNoProfile) {
		t.Fatalf("err=%v want ErrNoProfile", err)
	}
}

// TestLoadPrivateUnknownProfileError covers the case where the user requests
// a profile name that is not in the config file.
func TestLoadPrivateUnknownProfileError(t *testing.T) {
	isolate(t)
	if err := config.Save(&config.Config{
		Default:  "live",
		Profiles: map[string]config.Profile{"live": {ClientID: "id-live"}},
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{RequestedProfile: "ghost", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), `"ghost" not found`) {
		t.Fatalf("err=%v want profile-not-found", err)
	}
}

// TestLoadPrivateMissingCredentialWraps verifies that a credential miss is
// surfaced as a wrapped credential.ErrNotFound, so callers can distinguish
// "no profile" from "profile present but secret missing".
func TestLoadPrivateMissingCredentialWraps(t *testing.T) {
	isolate(t)
	if err := config.Save(&config.Config{
		Default:  "live",
		Profiles: map[string]config.Profile{"live": {ClientID: "id-live"}},
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Timeout: time.Second})
	if !errors.Is(err, credential.ErrNotFound) {
		t.Fatalf("err=%v want wrap of ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), `profile "live"`) {
		t.Errorf("err=%v missing profile name context", err)
	}
}

// TestLoadPrivateHappyPath returns the resolved profile, name, endpoint, and
// a non-nil client. RequestedProfile takes precedence over Default.
func TestLoadPrivateHappyPath(t *testing.T) {
	isolate(t)
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live": {ClientID: "id-live"},
			"test": {ClientID: "id-test"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.Default().Save("test", "secret-test"); err != nil {
		t.Fatal(err)
	}

	sess, err := Load(LoadOptions{RequestedProfile: "test", Timeout: time.Second})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sess.ProfileName != "test" {
		t.Errorf("ProfileName=%q want test", sess.ProfileName)
	}
	if sess.Profile == nil || sess.Profile.ClientID != "id-test" {
		t.Errorf("Profile=%+v want ClientID=id-test", sess.Profile)
	}
	if sess.Endpoint != config.DefaultEndpoint {
		t.Errorf("Endpoint=%q want %q", sess.Endpoint, config.DefaultEndpoint)
	}
	if sess.Client == nil {
		t.Fatal("Client must not be nil")
	}
}

// TestLoadPrivateRespectsEnvProfileFallback covers the empty-RequestedProfile
// path: config.Resolve falls through to E100X_PROFILE before Config.Default,
// and Load must not duplicate that lookup.
func TestLoadPrivateRespectsEnvProfileFallback(t *testing.T) {
	isolate(t)
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live": {ClientID: "id-live"},
			"test": {ClientID: "id-test"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.Default().Save("test", "secret-test"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("E100X_PROFILE", "test")

	sess, err := Load(LoadOptions{Timeout: time.Second})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sess.ProfileName != "test" {
		t.Errorf("ProfileName=%q want test (env override)", sess.ProfileName)
	}
}
