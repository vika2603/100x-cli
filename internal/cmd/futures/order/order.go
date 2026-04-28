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
		Use:     "order",
		Aliases: []string{"o"},
		Short:   "Limit/market order operations",
		Long: "Place, inspect, modify, and cancel futures orders.\n\n" +
			"Use `order place` for new orders, `order list` and `order show` to inspect existing\n" +
			"orders, `order edit` to rebook an open limit order, and `order cancel` / `cancel-all`\n" +
			"to remove open orders. Use `order deals` when you want private fill history.\n\n" +
			"Symbols are positional arguments. Order ids are also positional when a command acts on\n" +
			"a specific order.",
		Example: "# List open orders for BTCUSDT only\n" +
			"  100x futures order list --symbol BTCUSDT\n\n" +
			"# Place a BUY limit order on BTCUSDT at 70000 for size 0.001\n" +
			"  100x futures order place --limit --symbol BTCUSDT --side buy --size 0.001 --price 70000\n\n" +
			"# Cancel one open BTCUSDT order by id\n" +
			"  100x futures order cancel BTCUSDT <order-id>",
	}
	c.AddCommand(
		NewCmdPlace(f),
		NewCmdList(f),
		NewCmdShow(f),
		NewCmdEdit(f),
		NewCmdCancel(f),
		NewCmdCancelAll(f),
		NewCmdDeals(f),
	)
	return c
}
