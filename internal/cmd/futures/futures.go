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
	}
	c.AddCommand(
		order.NewCmdOrder(f),
		trigger.NewCmdTrigger(f),
		position.NewCmdPosition(f),
		balance.NewCmdBalance(f),
		market.NewCmdMarket(f),
	)
	return c
}
