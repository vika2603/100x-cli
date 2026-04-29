// Package futures groups the CLI's futures-product nouns under the cobra root.
package futures

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/balance"
	"github.com/vika2603/100x-cli/internal/cmd/futures/order"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger"
)

// NewCmdFutures returns the `100x futures` group.
func NewCmdFutures(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "futures",
		Aliases: []string{"f"},
		Short:   "Trade futures, manage positions, and read account state",
		Long: "Trade futures, manage positions, and read account state.\n\n" +
			"This group contains trading commands for orders, triggers, and positions, plus account\n" +
			"commands. Most write operations require a profile with private credentials. Public\n" +
			"market data lives under the top-level `market` command.\n\n" +
			"Symbols are usually positional arguments rather than flags. Start at the noun you care\n" +
			"about, then inspect the subcommands for read and write flows on that resource.",
		Example: "# List current account balances\n" +
			"  100x futures balance list\n\n" +
			"# List open BTCUSDT orders\n" +
			"  100x futures order list --symbol BTCUSDT\n\n" +
			"# Show the latest market state for BTCUSDT (lives under the top-level market command)\n" +
			"  100x market state BTCUSDT",
	}
	c.AddGroup(
		&cobra.Group{ID: "trade", Title: "Trading Commands"},
		&cobra.Group{ID: "account", Title: "Account Commands"},
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
	c.AddCommand(
		orderCmd,
		triggerCmd,
		positionCmd,
		preferenceCmd,
		balanceCmd,
	)
	factory.RequirePrivate(c)
	return c
}
