package profile

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
)

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
