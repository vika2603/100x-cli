// Package order wires the `100x futures order` cobra verbs.
//
// Each verb file follows the gh / kubectl pattern: an Options struct holds
// flag-bound state, NewCmdXxx assembles the cobra.Command, and runXxx is a
// pure function callable from tests without cobra. Cross-verb helpers live
// in shared/.
package order

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdOrder returns the `order` group.
func NewCmdOrder(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "order",
		Short: "Limit/market order operations",
	}
	c.AddCommand(
		NewCmdPlace(f),
		NewCmdList(f),
		NewCmdShow(f),
		NewCmdEdit(f),
		NewCmdCancel(f),
		NewCmdCancelAll(f),
	)
	return c
}
