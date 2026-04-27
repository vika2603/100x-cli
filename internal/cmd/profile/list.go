package profile

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/config"
)

func newCmdList(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured profiles",
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
				if len(rows) == 0 {
					return f.IO.Emptyln("No profiles configured.")
				}
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
