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
		Use:   "profile",
		Short: "Manage credential profiles",
	}
	c.AddCommand(newCmdAdd(), newCmdList(), newCmdUse(), newCmdShow(), newCmdRemove(f))
	return c
}
