package market

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/output"
)

func newCmdList(f *factory.Factory) *cobra.Command {
	var includeUnavailable bool
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all tradable instruments",
		Long: "List futures markets known to the gateway.\n\n" +
			"By default the command shows only currently tradable markets. Use\n" +
			"--include-unavailable to include markets that exist but are not currently available.",
		Example: "# List only markets that are currently tradable\n" +
			"  100x futures market list\n\n" +
			"# Include paused or unavailable markets in the result\n" +
			"  100x futures market list --include-unavailable\n\n" +
			"# Extract symbol, tick size, and availability as JSON\n" +
			"  100x --json futures market list --include-unavailable --jq 'map({symbol: .name, tick_size: .tick_size, available})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMarketList(cmd.Context(), f, includeUnavailable)
		},
	}
	c.Flags().BoolVar(&includeUnavailable, "include-unavailable", false, "include markets that are not currently tradable")
	return c
}

func runMarketList(ctx context.Context, f *factory.Factory, includeUnavailable bool) error {
	resp, err := f.Client.Market.MarketList(ctx, futures.MarketListReq{})
	if err != nil {
		return err
	}
	if resp == nil {
		resp = []futures.MarketItem{}
	}
	if !includeUnavailable {
		filtered := resp[:0]
		for _, m := range resp {
			if m.Available {
				filtered = append(filtered, m)
			}
		}
		resp = filtered
	}
	return f.IO.Render(resp, func() error {
		rows := make([][]string, 0, len(resp))
		for _, m := range resp {
			rows = append(rows, []string{m.Name, m.Stock, m.Money, m.TickSize, m.MakerFee, m.TakerFee})
		}
		return f.IO.Table([]output.Column{
			output.LCol("Symbol"), output.LCol("Base"), output.LCol("Quote"),
			output.RCol("Tick Size"), output.RCol("Maker Fee"), output.RCol("Taker Fee"),
		}, rows)
	})
}
