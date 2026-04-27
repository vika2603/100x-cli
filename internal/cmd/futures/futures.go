// Package futures groups the CLI's futures-product nouns under the cobra root.
package futures

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/balance"
	"github.com/vika2603/100x-cli/internal/cmd/futures/market"
	"github.com/vika2603/100x-cli/internal/cmd/futures/order"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger"
)

// NewCmdFutures returns the `100x futures` group.
func NewCmdFutures(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "futures",
		Aliases: []string{"f"},
		Short:   "Futures-product commands",
		Long: "Operate on the futures product surface.\n\n" +
			"This group contains trading commands for orders, triggers, and positions, plus account\n" +
			"and market-data commands. Most write operations require a profile with private credentials;\n" +
			"the `market` subtree is public and can be used without a secret.\n\n" +
			"Symbols are usually positional arguments rather than flags. Start at the noun you care\n" +
			"about, then inspect the subcommands for read and write flows on that resource.",
		Example: "# Show the latest market state for BTCUSDT\n" +
			"  100x futures market state BTCUSDT\n\n" +
			"# Place a BUY limit order on BTCUSDT at 70000 for size 0.001\n" +
			"  100x futures order place BTCUSDT --side buy --price 70000 --size 0.001\n\n" +
			"# Set BTCUSDT leverage and margin mode before trading\n" +
			"  100x futures preference BTCUSDT --leverage 25 --mode CROSS",
	}
	c.AddGroup(
		&cobra.Group{ID: "trade", Title: "Trading"},
		&cobra.Group{ID: "account", Title: "Account"},
		&cobra.Group{ID: "market", Title: "Market Data"},
	)
	orderCmd := order.NewCmdOrder(f)
	orderCmd.GroupID = "trade"
	triggerCmd := trigger.NewCmdTrigger(f)
	triggerCmd.GroupID = "trade"
	positionCmd := position.NewCmdPosition(f)
	positionCmd.GroupID = "trade"
	preferenceCmd := position.NewCmdPreference(f)
	preferenceCmd.GroupID = "trade"
	balanceCmd := balance.NewCmdBalance(f)
	balanceCmd.GroupID = "account"
	marketCmd := market.NewCmdMarket(f)
	marketCmd.GroupID = "market"
	c.AddCommand(orderCmd, triggerCmd, positionCmd, preferenceCmd, balanceCmd, marketCmd)
	return c
}
