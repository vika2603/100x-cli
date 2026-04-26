// Package position wires the `100x futures position` cobra verbs.
//
// `preference` and `leverage` use shared/preference.go to merge partial
// updates with the gateway's current state, since POST /setting/preference
// requires both leverage and position-type fields together.
package position

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdPosition returns the `position` group.
func NewCmdPosition(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "position",
		Short: "Open-position operations",
	}
	c.AddCommand(
		NewCmdList(f),
		NewCmdClose(f),
		NewCmdAdd(f),
		NewCmdMargin(f),
		NewCmdPreference(f),
		NewCmdLeverage(f),
	)
	return c
}
