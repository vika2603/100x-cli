// Package profile wires the `100x profile` verbs (config + credential).
package profile

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdProfile returns the `profile` group.
//
// Profile commands run before any credentials exist. The Factory is
// passed in only to surface root-level flags such as -y / --yes that
// `profile remove` consults for the destructive-op contract.
func NewCmdProfile(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "profile",
		Aliases: []string{"prof"},
		Short:   "Manage credential profiles",
		Long: "Manage named credential profiles for private API access.\n\n" +
			"Each profile is a name plus a client ID and its secret. Secrets are kept outside the\n" +
			"profile listing and never printed.\n\n" +
			"Use `profile add` to create or update credentials, `profile use` to switch the default\n" +
			"profile, and `profile show` or `profile list` to inspect the current config state.",
		Example: "# Add a profile named test\n" +
			"  100x profile add test --client-id <CID>\n\n" +
			"# List every configured profile in a table\n" +
			"  100x profile list\n\n" +
			"# Show client ID and secret status for profile test\n" +
			"  100x profile show test",
	}
	c.AddCommand(newCmdAdd(f), newCmdList(f), newCmdCurrent(f), newCmdUse(f), newCmdShow(f), newCmdRemove(f))
	return c
}
