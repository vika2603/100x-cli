package profile

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/prompt"
)

// AddOptions captures the flag-bound state of `profile add`.
type AddOptions struct {
	Name       string
	ClientID   string
	ClientKey  string
	SetDefault bool
}

func newCmdAdd(f *factory.Factory) *cobra.Command {
	opts := &AddOptions{}
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Add or update a profile",
		Long: "Add or update one credential profile.\n\n" +
			"Each profile is a name plus a client ID and its client key. The client key is prompted for when --client-key is omitted and is never echoed back.",
		Example: "# Add profile test and prompt for its client key\n" +
			"  100x profile add test --client-id <CID>",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.Name = args[0]
			payload, err := runAdd(opts)
			if err != nil {
				return err
			}
			return f.IO.Render(payload, func() error {
				f.IO.Println("saved profile", payload.Name)
				return nil
			})
		},
	}
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "Client ID for this profile")
	c.Flags().StringVar(&opts.ClientKey, "client-key", "", "Client key; prompt when omitted")
	c.Flags().BoolVar(&opts.SetDefault, "default", false, "Make this the default profile")
	return c
}

func runAdd(opts *AddOptions) (profileDetail, error) {
	if err := validateProfileName(opts.Name); err != nil {
		return profileDetail{}, err
	}
	if err := fillAddInputs(opts); err != nil {
		return profileDetail{}, err
	}
	cfg, err := config.Load()
	if err != nil {
		return profileDetail{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	prev, hadPrev := cfg.Profiles[opts.Name]
	rollback, err := snapshotForRollback(opts.ClientID)
	if err != nil {
		return profileDetail{}, err
	}
	if err := credential.SaveSecret(opts.ClientID, credential.Envelope{
		ClientID: opts.ClientID, ClientKey: opts.ClientKey,
	}); err != nil {
		return profileDetail{}, err
	}
	cfg.Profiles[opts.Name] = config.Profile{ClientID: opts.ClientID}
	if opts.SetDefault || cfg.Default == "" {
		cfg.Default = opts.Name
	}
	if err := config.Save(cfg); err != nil {
		rollback()
		return profileDetail{}, err
	}
	// If this profile was rebound to a new client_id, drop the old entry
	// when no other profile still points at it. The previous entry is
	// reachable by its (user-known) client_id, so this is a tidy-up rather
	// than a recovery requirement.
	if hadPrev && prev.ClientID != "" && prev.ClientID != opts.ClientID &&
		!clientIDInUse(cfg, prev.ClientID) {
		_ = credential.DeleteSecret(prev.ClientID)
	}
	return profileDetail{
		Name: opts.Name, ClientID: opts.ClientID,
		Current: cfg.Default == opts.Name, SecretStored: true,
	}, nil
}

// snapshotForRollback captures the current secret stored under clientID
// (if any) and returns a closure that restores it. Used to undo a
// SaveSecret that was followed by a config.Save failure: a no-op on
// success, the only consumer is the failure branch.
func snapshotForRollback(clientID string) (func(), error) {
	prior, err := credential.LoadSecret(clientID)
	switch {
	case err == nil:
		return func() { _ = credential.SaveSecret(clientID, prior) }, nil
	case errors.Is(err, credential.ErrNotFound):
		return func() { _ = credential.DeleteSecret(clientID) }, nil
	default:
		return nil, err
	}
}

// clientIDInUse reports whether any profile currently in cfg references
// the given client_id. Two profiles may legitimately share one client_id
// (the API identity) and therefore one secret.
func clientIDInUse(cfg *config.Config, clientID string) bool {
	for _, p := range cfg.Profiles {
		if p.ClientID == clientID {
			return true
		}
	}
	return false
}

func fillAddInputs(opts *AddOptions) error {
	var err error
	if opts.ClientID == "" {
		opts.ClientID, err = promptInput("Client ID", "client-id", "")
		if err != nil {
			return err
		}
	}
	if opts.ClientID == "" {
		return errors.New("profile add: client-id is required")
	}
	if opts.ClientKey == "" {
		opts.ClientKey, err = prompt.Secret("Client key")
		if errors.Is(err, prompt.ErrNoTTY) {
			return errors.New("profile add: --client-key is required when stdin is not a terminal")
		}
		if err != nil {
			return err
		}
	}
	if opts.ClientKey == "" {
		return errors.New("profile add: client-key is required")
	}
	return nil
}

func promptInput(title, flagName, placeholder string) (string, error) {
	value, err := prompt.Input(title, placeholder)
	if errors.Is(err, prompt.ErrNoTTY) {
		return "", fmt.Errorf("profile add: --%s is required in non-interactive mode", flagName)
	}
	return value, err
}
