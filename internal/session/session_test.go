package session

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/vika2603/100x-cli/api/futures"
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
	t.Setenv("E100X_ENDPOINT", "https://api.example.com/")
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
// a non-nil client. RequestedProfile takes precedence over Default. The
// endpoint comes from $E100X_ENDPOINT.
func TestLoadPrivateHappyPath(t *testing.T) {
	isolate(t)
	t.Setenv("E100X_ENDPOINT", "https://api.example.com/")
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live": {ClientID: "id-live"},
			"test": {ClientID: "id-test"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-test", credential.Envelope{
		ClientID: "id-test", ClientKey: "secret-test",
	}); err != nil {
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
	if sess.Endpoint != "https://api.example.com/" {
		t.Errorf("Endpoint=%q want https://api.example.com/", sess.Endpoint)
	}
	if sess.Client == nil {
		t.Fatal("Client must not be nil")
	}
}

// TestLoadPrivateMissingEndpointReturnsErrNoEndpoint covers the new failure
// mode: a profile alone is no longer enough; without env or build-time
// default the session cannot be constructed.
func TestLoadPrivateMissingEndpointReturnsErrNoEndpoint(t *testing.T) {
	isolate(t)
	saved := config.DefaultEndpoint
	t.Cleanup(func() { config.DefaultEndpoint = saved })
	config.DefaultEndpoint = ""
	if err := config.Save(&config.Config{
		Default:  "live",
		Profiles: map[string]config.Profile{"live": {ClientID: "id-live"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-live", credential.Envelope{
		ClientID: "id-live", ClientKey: "secret-live",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Timeout: time.Second})
	if !errors.Is(err, config.ErrNoEndpoint) {
		t.Fatalf("err=%v want ErrNoEndpoint", err)
	}
}

// TestLoadPublicMissingEndpointReturnsErrNoEndpoint covers the public path,
// which also depends on a configured endpoint.
func TestLoadPublicMissingEndpointReturnsErrNoEndpoint(t *testing.T) {
	isolate(t)
	saved := config.DefaultEndpoint
	t.Cleanup(func() { config.DefaultEndpoint = saved })
	config.DefaultEndpoint = ""

	_, err := Load(LoadOptions{Public: true, Timeout: time.Second})
	if !errors.Is(err, config.ErrNoEndpoint) {
		t.Fatalf("err=%v want ErrNoEndpoint", err)
	}
}

// TestLoadAppliesPerRequestTimeout verifies that LoadOptions.Timeout is wired
// through to the underlying http.Client.Timeout so each HTTP attempt is
// bounded independently. A fast-failing timeout against a hanging server is
// the cheapest behavioural proof that the field is no longer dead.
func TestLoadAppliesPerRequestTimeout(t *testing.T) {
	isolate(t)

	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-hang:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("E100X_ENDPOINT", srv.URL)

	sess, err := Load(LoadOptions{Public: true, Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	ctx := futures.WithRetryPolicy(context.Background(), futures.NoRetry)
	start := time.Now()
	_, err = sess.Client.Market.MarketList(ctx, futures.MarketListReq{})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("MarketList must fail when the server hangs")
	}
	if elapsed > time.Second {
		t.Errorf("elapsed=%v want <1s; per-request timeout did not fire", elapsed)
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		t.Errorf("err=%v want a net.Error with Timeout()=true", err)
	}
}

// TestLoadDefaultsToBackstopWhenTimeoutZero covers the disable path: passing
// Timeout=0 leaves the client free of a per-request cap from the flag, so a
// short-running server completes normally instead of being cut off.
func TestLoadDefaultsToBackstopWhenTimeoutZero(t *testing.T) {
	isolate(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":[]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("E100X_ENDPOINT", srv.URL)

	sess, err := Load(LoadOptions{Public: true, Timeout: 0})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	ctx := futures.WithRetryPolicy(context.Background(), futures.NoRetry)
	if _, err := sess.Client.Market.MarketList(ctx, futures.MarketListReq{}); err != nil {
		t.Fatalf("MarketList with Timeout=0: %v", err)
	}
}

// TestLoadPrivateRespectsEnvProfileFallback covers the empty-RequestedProfile
// path: config.Resolve falls through to E100X_PROFILE before Config.Default,
// and Load must not duplicate that lookup.
func TestLoadPrivateRespectsEnvProfileFallback(t *testing.T) {
	isolate(t)
	t.Setenv("E100X_ENDPOINT", "https://api.example.com/")
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live": {ClientID: "id-live"},
			"test": {ClientID: "id-test"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-test", credential.Envelope{
		ClientID: "id-test", ClientKey: "secret-test",
	}); err != nil {
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
