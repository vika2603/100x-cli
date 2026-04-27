package market

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

func newCmdDeals(f *factory.Factory) *cobra.Command {
	var symbol string
	var limit int
	c := &cobra.Command{
		Use:   "deals <symbol>",
		Short: "List the latest public trades",
		Long: "List the latest public trades for one symbol.\n\n" +
			"This is market-wide trade flow, not your private fills. Use `order deals` when you want\n" +
			"your account's fill history instead of the public tape.\n\n" +
			"The gateway caps responses at 50 trades; --limit values higher than that have no\n" +
			"additional effect.",
		Example: "# Show the latest public trades for BTCUSDT\n" +
			"  100x futures market deals BTCUSDT\n\n" +
			"# Show the latest 50 public trades for BTCUSDT\n" +
			"  100x futures market deals BTCUSDT --limit 50\n\n" +
			"# Extract trade id, side, price, and size as JSON\n" +
			"  100x --json futures market deals BTCUSDT --limit 20 --jq 'map({id, type, price, volume})'",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol = args[0]
			if err := clierr.PositiveInt("--limit", limit); err != nil {
				return err
			}
			resp, err := f.Client.Market.MarketDeals(cmd.Context(), futures.MarketDealsReq{Market: symbol})
			if err != nil {
				return err
			}
			if resp == nil {
				resp = []futures.MarketDealItem{}
			}
			resp = limitSlice(resp, limit)
			return f.IO.Render(resp, func() error { return printMarketDeals(f.IO, resp) })
		},
	}
	c.Flags().IntVar(&limit, "limit", 20, "recent trades to show (server caps at 50)")
	return c
}

func printMarketDeals(io *output.Renderer, rows []futures.MarketDealItem) error {
	if len(rows) == 0 {
		return io.Emptyln("No public trades found.")
	}
	out := make([][]string, 0, len(rows))
	for _, d := range rows {
		out = append(out, []string{
			strconv.Itoa(d.ID),
			format.Enum(d.Type),
			d.Price,
			d.Volume,
			format.UnixMillis(d.Time),
		})
	}
	return io.Table([]output.Column{
		output.LCol("Trade ID"), output.LCol("Side"),
		output.RCol("Price"), output.RCol("Size"),
		output.LCol("Time"),
	}, out)
}
