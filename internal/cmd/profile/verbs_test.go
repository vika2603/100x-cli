package profile

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/output"
)

func TestListEmptyHuman(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stdout := &bytes.Buffer{}
	cmd := newCmdList(&factory.Factory{
		IO: &output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No profiles configured.") {
		t.Fatalf("stdout=%q", got)
	}
}

func isolateProfile(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("E100X_PROFILE", "")
	keyring.MockInit()
}

// TestAddRebindsClientIDDropsOldSecret: changing a profile's client_id must
// store the new secret under the new client_id and remove the old entry
// when nothing else references it. The old client_id is user-known, so a
// failure to clean up is a tidy-up issue, not a recovery requirement.
func TestAddRebindsClientIDDropsOldSecret(t *testing.T) {
	isolateProfile(t)
	if err := config.Save(&config.Config{
		Default:  "live",
		Profiles: map[string]config.Profile{"live": {ClientID: "id-old"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-old", credential.Envelope{
		ClientID: "id-old", Secret: "old-secret",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := runAdd(&AddOptions{
		Name:     "live",
		ClientID: "id-new",
		Secret:   "new-secret",
	}); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Profiles["live"].ClientID; got != "id-new" {
		t.Errorf("ClientID=%q want id-new", got)
	}
	env, err := credential.LoadSecret("id-new")
	if err != nil {
		t.Fatalf("LoadSecret(id-new): %v", err)
	}
	if env.Secret != "new-secret" {
		t.Errorf("envelope=%+v want secret=new-secret", env)
	}
	if _, err := credential.LoadSecret("id-old"); !errors.Is(err, credential.ErrNotFound) {
		t.Errorf("old secret survived: err=%v", err)
	}
}

// TestAddSharedClientIDKeepsOldSecret: when a profile is rebound to a
// client_id used by another profile, the previous client_id entry must
// stay because some other profile still owns it.
func TestAddRebindKeepsOtherProfilesSecret(t *testing.T) {
	isolateProfile(t)
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live":  {ClientID: "id-old"},
			"other": {ClientID: "id-old"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-old", credential.Envelope{
		ClientID: "id-old", Secret: "old-secret",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := runAdd(&AddOptions{
		Name:     "live",
		ClientID: "id-new",
		Secret:   "new-secret",
	}); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	if _, err := credential.LoadSecret("id-old"); err != nil {
		t.Errorf("shared old secret deleted while sibling still references it: %v", err)
	}
}

func TestAddRestoresClientIDSecretWhenConfigSaveFails(t *testing.T) {
	isolateProfile(t)
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live":  {ClientID: "id-live"},
			"other": {ClientID: "id-shared"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-shared", credential.Envelope{
		ClientID: "id-shared", Secret: "old-secret",
	}); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(config.Dir(), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(config.Dir(), 0o700) })

	_, err := runAdd(&AddOptions{
		Name:     "live",
		ClientID: "id-shared",
		Secret:   "new-secret",
	})
	if err == nil {
		t.Fatal("runAdd succeeded despite unwritable config dir")
	}

	env, err := credential.LoadSecret("id-shared")
	if err != nil {
		t.Fatalf("LoadSecret(id-shared): %v", err)
	}
	if env.Secret != "old-secret" {
		t.Fatalf("secret=%q want restored old-secret", env.Secret)
	}
}

// TestRemoveDeletesSecretBeforeConfig: the secret is removed before
// config.toml is rewritten, so a DeleteSecret failure aborts cleanly with
// the profile still present and retryable.
func TestRemoveDeletesSecretBeforeConfig(t *testing.T) {
	isolateProfile(t)
	if err := config.Save(&config.Config{
		Default:  "live",
		Profiles: map[string]config.Profile{"live": {ClientID: "id-live"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-live", credential.Envelope{
		ClientID: "id-live", Secret: "secret",
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdRemove(&factory.Factory{
		IO:  &output.Renderer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Format: output.FormatJSON},
		Yes: true,
	})
	cmd.SetArgs([]string{"live"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("remove: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Profiles["live"]; ok {
		t.Error("profile not removed from config")
	}
	if _, err := credential.LoadSecret("id-live"); !errors.Is(err, credential.ErrNotFound) {
		t.Errorf("blob not deleted: err=%v", err)
	}
}

// TestRemoveSharedClientIDKeepsBlob: removing one of two profiles that
// share a client_id must not delete the underlying secret, since the
// sibling still needs it.
func TestRemoveSharedClientIDKeepsBlob(t *testing.T) {
	isolateProfile(t)
	if err := config.Save(&config.Config{
		Default: "live",
		Profiles: map[string]config.Profile{
			"live":     {ClientID: "id-shared"},
			"readonly": {ClientID: "id-shared"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := credential.SaveSecret("id-shared", credential.Envelope{
		ClientID: "id-shared", Secret: "secret",
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdRemove(&factory.Factory{
		IO:  &output.Renderer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Format: output.FormatJSON},
		Yes: true,
	})
	cmd.SetArgs([]string{"live"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if _, err := credential.LoadSecret("id-shared"); err != nil {
		t.Errorf("shared blob deleted while sibling still references it: %v", err)
	}
}
