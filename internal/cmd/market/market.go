// Package market wires the `100x market` verbs.
package market

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdMarket returns the `market` group.
func NewCmdMarket(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "market",
		Aliases: []string{"m"},
		Short:   "Public market data",
		Long: "Read public market data.\n\n" +
			"These commands do not place trades or require private credentials. Use them to discover\n" +
			"symbols, inspect ticker state, read the current order book, list recent public trades,\n" +
			"or query candle history for charting and diagnostics.",
		Example: "# Show the latest ticker snapshot for BTCUSDT\n" +
			"  100x market state BTCUSDT\n\n" +
			"# Show the top 5 bid and ask levels for BTCUSDT\n" +
			"  100x market depth BTCUSDT --limit 5",
	}
	c.AddCommand(newCmdList(f), newCmdState(f), newCmdDepth(f), newCmdDeals(f), newCmdKline(f))
	// Market endpoints are public and do not need a signed client.
	factory.RequirePublic(c)
	return c
}
