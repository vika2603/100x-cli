// Package balance wires the `100x futures balance` verbs.
package balance

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdBalance returns the `balance` group.
func NewCmdBalance(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "balance",
		Aliases: []string{"bal"},
		Short:   "Wallet balance and asset history",
		Long: "Inspect wallet balances and asset movement history.\n\n" +
			"Use `balance list` for the current account snapshot and `balance history` for paginated\n" +
			"asset changes such as deposits, withdrawals, and faucet activity.",
		Example: "# Show the current wallet balance snapshot for every asset\n" +
			"  100x futures balance list\n\n" +
			"# Review paginated asset history for USDT\n" +
			"  100x futures balance history --currency USDT --page-size 20",
	}
	c.AddCommand(newCmdList(f), newCmdHistory(f))
	return c
}
