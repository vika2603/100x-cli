package profile

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/prompt"
)

var (
	profileNameRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
	// reservedProfileNames is the closed set of tokens the CLI keeps for
	// future use. Keeping `default`, `current`, etc. unowned means we can
	// later use them to mean "the active profile" without breaking
	// existing config.
	reservedProfileNames = map[string]struct{}{
		"default": {}, "current": {}, "me": {}, "self": {}, "all": {}, "none": {},
	}
)

func validateProfileName(name string) error {
	if name == "" {
		return errors.New("profile name is required")
	}
	if len(name) > 32 {
		return fmt.Errorf("profile name %q is longer than 32 chars", name)
	}
	if !profileNameRE.MatchString(name) {
		return fmt.Errorf("profile name %q must match [a-z0-9_-]+", name)
	}
	if _, reserved := reservedProfileNames[name]; reserved {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	return nil
}

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

type profileListItem struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id"`
	Current  bool   `json:"current"`
}

type currentProfile struct {
	Name string `json:"name"`
}

type profileDetail struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	Current      bool   `json:"current"`
	SecretStored bool   `json:"secret_stored"`
}

func newCmdList(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		Example: "# List all profiles in a human-readable table\n" +
			"  100x profile list\n\n" +
			"# List all profiles as JSON for scripts\n" +
			"  100x --json profile list\n\n" +
			"# Extract only the current profile and client ID\n" +
			"  100x --json profile list --jq '.[] | select(.current) | {name, client_id}'",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Profiles))
			for n := range cfg.Profiles {
				names = append(names, n)
			}
			sort.Strings(names)
			rows := make([]profileListItem, 0, len(names))
			for _, n := range names {
				p := cfg.Profiles[n]
				rows = append(rows, profileListItem{
					Name: n, ClientID: p.ClientID, Current: n == cfg.Default,
				})
			}
			return f.IO.Render(rows, func() error {
				out := make([][]string, 0, len(rows))
				for _, r := range rows {
					current := ""
					if r.Current {
						current = "*"
					}
					out = append(out, []string{r.Name, r.ClientID, current})
				}
				return f.IO.Table([]string{"Name", "Client ID", "Current"}, out)
			})
		},
	}
}

func newCmdCurrent(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Print the current profile",
		Example: "# Print the active default profile name\n" +
			"  100x profile current",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.Default == "" {
				return config.ErrNoProfile
			}
			if _, ok := cfg.Profiles[cfg.Default]; !ok {
				return fmt.Errorf("profile %q not found", cfg.Default)
			}
			payload := currentProfile{Name: cfg.Default}
			return f.IO.Render(payload, func() error {
				return f.IO.Resultln(payload.Name)
			})
		},
	}
}

func newCmdShow(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show one profile (secret redacted)",
		Example: "# Show profile test with its client ID and secret status\n" +
			"  100x profile show test",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, ok := cfg.Profiles[args[0]]
			if !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			payload := profileDetail{
				Name: args[0], ClientID: p.ClientID,
				Current: args[0] == cfg.Default, SecretStored: true,
			}
			return f.IO.Render(payload, func() error {
				return f.IO.Object([]output.KV{
					{Key: "Name", Value: payload.Name},
					{Key: "Client ID", Value: payload.ClientID},
					{Key: "Current", Value: fmt.Sprint(payload.Current)},
					{Key: "Secret", Value: "<stored>"},
				})
			})
		},
	}
}

func newCmdUse(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default profile",
		Example: "# Make one profile the default for future commands\n" +
			"  100x profile use test",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			cfg.Default = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			payload := currentProfile{Name: args[0]}
			return f.IO.Render(payload, func() error {
				return f.IO.Resultln(payload.Name)
			})
		},
	}
}

// completeProfileNames lists configured profile names for tab completion
// without making any network call.
func completeProfileNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		out = append(out, name)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func newCmdRemove(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a profile (and its secret)",
		Example: "# Remove one profile with confirmation\n" +
			"  100x profile remove test\n\n" +
			"# Remove one profile without the confirmation prompt\n" +
			"  100x profile remove test --yes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			ok, err := prompt.ConfirmDestructive(
				fmt.Sprintf("Delete profile %q and its stored secret?", args[0]), f.Yes)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			delete(cfg.Profiles, args[0])
			if cfg.Default == args[0] {
				cfg.Default = ""
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			return credential.Default().Delete(args[0])
		},
	}
}
