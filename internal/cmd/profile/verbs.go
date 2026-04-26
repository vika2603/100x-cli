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

func newCmdAdd(f *factory.Factory) *cobra.Command {
	opts := &AddOptions{}
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Add or update a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.Name = args[0]
			if err := runAdd(opts); err != nil {
				return err
			}
			payload := currentProfile{Name: opts.Name}
			return f.IO.Render(payload, func() error {
				f.IO.Println("saved profile", opts.Name)
				return nil
			})
		},
	}
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
	return nil
}

type profileListItem struct {
	Name     string `json:"name"`
	Env      string `json:"env"`
	Endpoint string `json:"endpoint"`
	Current  bool   `json:"current"`
}

type currentProfile struct {
	Name string `json:"name"`
}

type profileDetail struct {
	Name         string `json:"name"`
	Endpoint     string `json:"endpoint"`
	ClientID     string `json:"client_id"`
	Env          string `json:"env"`
	Current      bool   `json:"current"`
	SecretStored bool   `json:"secret_stored"`
}

func newCmdList(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
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
					Name: n, Env: p.Env, Endpoint: p.Endpoint, Current: n == cfg.Default,
				})
			}
			return f.IO.Render(rows, func() error {
				out := make([][]string, 0, len(rows))
				for _, r := range rows {
					current := ""
					if r.Current {
						current = "*"
					}
					out = append(out, []string{r.Name, r.Env, r.Endpoint, current})
				}
				return f.IO.Table([]string{"Name", "Env", "Endpoint", "Current"}, out)
			})
		},
	}
}

func newCmdCurrent(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Print the current profile",
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
		Use:               "show <name>",
		Short:             "Show one profile (secret redacted)",
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
				Name: args[0], Endpoint: p.Endpoint, ClientID: p.ClientID, Env: p.Env,
				Current: args[0] == cfg.Default, SecretStored: true,
			}
			return f.IO.Render(payload, func() error {
				return f.IO.Object([]output.KV{
					{Key: "Name", Value: payload.Name},
					{Key: "Endpoint", Value: payload.Endpoint},
					{Key: "Client ID", Value: payload.ClientID},
					{Key: "Env", Value: payload.Env},
					{Key: "Current", Value: fmt.Sprint(payload.Current)},
					{Key: "Secret", Value: "<stored>"},
				})
			})
		},
	}
}

func newCmdUse(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:               "use <name>",
		Short:             "Set the default profile",
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
		Use:               "remove <name>",
		Short:             "Delete a profile (and its secret)",
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
