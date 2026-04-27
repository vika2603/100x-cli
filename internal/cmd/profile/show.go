package profile

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/output"
)

func newCmdShow(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show one profile (secret redacted)",
		Example: "# Show profile test with its client ID and secret status\n" +
			"  100x profile show test",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: CompleteNames,
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
