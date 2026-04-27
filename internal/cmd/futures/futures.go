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
			"# List current account balances\n" +
			"  100x futures balance list\n\n" +
			"# List open BTCUSDT orders\n" +
			"  100x futures order list --symbol BTCUSDT",
	}
	c.AddGroup(
		&cobra.Group{ID: "trade", Title: "Trading Commands"},
		&cobra.Group{ID: "account", Title: "Account Commands"},
		&cobra.Group{ID: "market", Title: "Market Data Commands"},
	)
	orderCmd := order.NewCmdOrder(f)
	orderCmd.GroupID = "trade"
	ordersCmd := order.NewCmdOrders(f)
	ordersCmd.GroupID = "trade"
	triggerCmd := trigger.NewCmdTrigger(f)
	triggerCmd.GroupID = "trade"
	triggersCmd := trigger.NewCmdTriggers(f)
	triggersCmd.GroupID = "trade"
	positionCmd := position.NewCmdPosition(f)
	positionCmd.GroupID = "trade"
	positionsCmd := position.NewCmdPositions(f)
	positionsCmd.GroupID = "trade"
	preferenceCmd := position.NewCmdPreference(f)
	preferenceCmd.GroupID = "trade"
	balanceCmd := balance.NewCmdBalance(f)
	balanceCmd.GroupID = "account"
	balancesCmd := balance.NewCmdBalances(f)
	balancesCmd.GroupID = "account"
	marketCmd := market.NewCmdMarket(f)
	marketCmd.GroupID = "market"
	c.AddCommand(
		orderCmd, ordersCmd,
		triggerCmd, triggersCmd,
		positionCmd, positionsCmd,
		preferenceCmd,
		balanceCmd, balancesCmd,
		marketCmd,
	)
	// Default for the futures subtree: signed private client. The market
	// child group overrides this with RequirePublic for its own descendants.
	factory.RequirePrivate(c)
	return c
}
