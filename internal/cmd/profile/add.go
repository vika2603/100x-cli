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
	Secret     string
	SetDefault bool
}

func newCmdAdd(f *factory.Factory) *cobra.Command {
	opts := &AddOptions{}
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Add or update a profile",
		Long: "Add or update one credential profile.\n\n" +
			"Profiles store client identity. The secret is saved in the OS keychain. The API endpoint is built into the CLI; use E100X_ENDPOINT to override it for one command.",
		Example: "# Add profile test and store its API secret in the keychain\n" +
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
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "gateway client ID for this profile")
	c.Flags().StringVar(&opts.Secret, "secret", "", "gateway client secret; prompt when omitted")
	c.Flags().BoolVar(&opts.SetDefault, "default", false, "make this the default profile")
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
	cfg.Profiles[opts.Name] = config.Profile{ClientID: opts.ClientID}
	if opts.SetDefault || cfg.Default == "" {
		cfg.Default = opts.Name
	}
	if err := config.Save(cfg); err != nil {
		return profileDetail{}, err
	}
	if err := credential.Default().Save(opts.Name, opts.Secret); err != nil {
		return profileDetail{}, err
	}
	return profileDetail{
		Name: opts.Name, ClientID: opts.ClientID,
		Current: cfg.Default == opts.Name, SecretStored: true,
	}, nil
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
	if opts.Secret == "" {
		opts.Secret, err = prompt.Secret("API secret")
		if errors.Is(err, prompt.ErrNoTTY) {
			return errors.New("profile add: --secret is required in non-interactive mode")
		}
		if err != nil {
			return err
		}
	}
	if opts.Secret == "" {
		return errors.New("profile add: secret is required")
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
