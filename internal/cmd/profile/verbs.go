package profile

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
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
		return errors.New("--name is required")
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
	Endpoint   string
	ClientID   string
	Env        string
	Secret     string
	SetDefault bool
}

func newCmdAdd() *cobra.Command {
	opts := &AddOptions{}
	c := &cobra.Command{
		Use:   "add",
		Short: "Add or update a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAdd(opts)
		},
	}
	c.Flags().StringVar(&opts.Name, "name", "", "profile name")
	c.Flags().StringVar(&opts.Endpoint, "endpoint", "", "API endpoint (e.g. https://api.example.com)")
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "client_id issued by the gateway")
	c.Flags().StringVar(&opts.Env, "env", "live", "free-text env label (live | test | paper)")
	c.Flags().StringVar(&opts.Secret, "secret", "", "API secret (omit to be prompted)")
	c.Flags().BoolVar(&opts.SetDefault, "default", false, "make this the default profile")
	return c
}

func runAdd(opts *AddOptions) error {
	if err := validateProfileName(opts.Name); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	secret := opts.Secret
	if secret == "" {
		secret, err = prompt.Secret("API secret")
		if err != nil {
			return err
		}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	cfg.Profiles[opts.Name] = config.Profile{Endpoint: opts.Endpoint, ClientID: opts.ClientID, Env: opts.Env}
	if opts.SetDefault || cfg.Default == "" {
		cfg.Default = opts.Name
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	if err := credential.Default().Save(opts.Name, secret); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "saved profile", opts.Name)
	return nil
}

func newCmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			for n, p := range cfg.Profiles {
				marker := ""
				if n == cfg.Default {
					marker = " (default)"
				}
				fmt.Printf("%s\t%s\t%s%s\n", n, p.Env, p.Endpoint, marker)
			}
			return nil
		},
	}
}

func newCmdUse() *cobra.Command {
	return &cobra.Command{
		Use:               "use <name>",
		Short:             "Set the default profile",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			cfg.Default = args[0]
			return config.Save(cfg)
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

func newCmdShow() *cobra.Command {
	return &cobra.Command{
		Use:               "show <name>",
		Short:             "Show one profile (secret redacted)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, ok := cfg.Profiles[args[0]]
			if !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}
			fmt.Printf("name: %s\nendpoint: %s\nclient_id: %s\nenv: %s\nsecret: <stored>\n", args[0], p.Endpoint, p.ClientID, p.Env)
			return nil
		},
	}
}

func newCmdRemove(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:               "remove <name>",
		Short:             "Delete a profile (and its secret)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
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
