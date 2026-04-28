package market

import (
	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/output"
)

// DepthOptions for `market depth`.
type DepthOptions struct {
	Symbol   string
	TickSize string
	Limit    int

	Factory *factory.Factory
}

func newCmdDepth(f *factory.Factory) *cobra.Command {
	opts := &DepthOptions{Factory: f}
	c := &cobra.Command{
		Use:   "depth <symbol>",
		Short: "Show the order-book snapshot",
		Long: "Show the current order-book snapshot for one symbol.\n\n" +
			"Use --limit to control how many bid and ask levels are shown. Use --tick-size to request\n" +
			"merged book levels when you want a less granular view of the market depth.\n\n" +
			"Responses contain at most 50 levels per side; --limit values higher than that have no\n" +
			"additional effect, and a less liquid market may return fewer levels regardless of --limit.",
		Example: "# Show the current order book for BTCUSDT\n" +
			"  100x futures market depth BTCUSDT\n\n" +
			"# Merge levels by tick size 0.1 and show 20 bids and 20 asks\n" +
			"  100x futures market depth BTCUSDT --tick-size 0.1 --limit 20\n\n" +
			"# Extract the best bid and ask price as JSON\n" +
			"  100x --json futures market depth BTCUSDT --limit 1 --jq '{best_bid: .bids[0].price, best_ask: .asks[0].price}'",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			if err := clierr.PositiveInt("--limit", opts.Limit); err != nil {
				return err
			}
			resp, err := f.Client.Market.MarketDepth(cmd.Context(), futures.MarketDepthReq{Market: opts.Symbol, Merge: opts.TickSize})
			if err != nil {
				return err
			}
			trimDepth(resp, opts.Limit)
			return f.IO.Render(resp, func() error { return printDepth(f.IO, resp) })
		},
	}
	c.Flags().StringVar(&opts.TickSize, "tick-size", "", "Merge book levels by this tick size")
	c.Flags().IntVar(&opts.Limit, "limit", 10, "Levels to show on each side (server caps at 50)")
	return c
}

func trimDepth(resp *futures.MarketDepthResp, limit int) {
	resp.Asks = limitSlice(resp.Asks, limit)
	resp.Bids = limitSlice(resp.Bids, limit)
}

func printDepth(io *output.Renderer, d *futures.MarketDepthResp) error {
	if len(d.Asks) == 0 && len(d.Bids) == 0 {
		return io.Emptyln("No depth levels found.")
	}
	rows := make([][]string, 0, len(d.Asks)+len(d.Bids))
	for _, ask := range d.Asks {
		rows = append(rows, []string{"ASK", ask.Price, ask.Volume})
	}
	for _, bid := range d.Bids {
		rows = append(rows, []string{"BID", bid.Price, bid.Volume})
	}
	return io.Table([]output.Column{
		output.LCol("Side"), output.RCol("Price"), output.RCol("Size"),
	}, rows)
}
