// Package position wires the `100x futures position` cobra verbs.
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
		Long: "Inspect and manage open positions.\n\n" +
			"Use `position list` and `position history` to inspect current and closed positions.\n" +
			"Use `position add` and `position close` to trade against an existing position. Use\n" +
			"`position margin` to inspect or adjust isolated margin.\n\n" +
			"When a command needs a specific position, pass --position-id or provide a symbol that\n" +
			"resolves to exactly one matching position.",
		Example: "# List open positions for BTCUSDT only\n" +
			"  100x futures position list --symbol BTCUSDT\n\n" +
			"# Add 10 units of isolated margin to a BTCUSDT position\n" +
			"  100x futures position margin BTCUSDT --position-id <position-id> --add 10\n\n" +
			"# Close the full BTCUSDT position at market without the prompt\n" +
			"  100x futures position close BTCUSDT --position-id <position-id> --type market --yes",
	}
	c.AddCommand(
		NewCmdList(f),
		NewCmdHistory(f),
		NewCmdClose(f),
		NewCmdAdd(f),
		NewCmdMargin(f),
	)
	return c
}
