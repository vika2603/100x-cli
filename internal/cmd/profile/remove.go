package profile

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
	"github.com/vika2603/100x-cli/internal/prompt"
)

func newCmdRemove(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a profile (and its secret)",
		Example: "# Remove one profile with confirmation\n" +
			"  100x profile remove test\n\n" +
			"# Remove one profile without the confirmation prompt\n" +
			"  100x profile remove test --yes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: CompleteNames,
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
			if err := credential.Default().Delete(args[0]); err != nil {
				return err
			}
			payload := currentProfile{Name: args[0]}
			return f.IO.Render(payload, func() error {
				return f.IO.Resultln("removed profile", args[0])
			})
		},
	}
}
