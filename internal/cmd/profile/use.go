package profile

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
)

func newCmdUse(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default profile",
		Example: "# Make one profile the default for future commands\n" +
			"  100x profile use test",
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
