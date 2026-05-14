package market

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/wire"
)

// StateOptions for `market state`.
type StateOptions struct {
	Symbol string

	Factory *factory.Factory
}

func newCmdState(f *factory.Factory) *cobra.Command {
	opts := &StateOptions{Factory: f}
	c := &cobra.Command{
		Use:   "state [symbol]",
		Short: "Show ticker state for one or all markets",
		Long: "Show ticker-style market state.\n\n" +
			"Without a symbol, the command prints a table for every market. With a symbol, it prints\n" +
			"one detailed record including last price, index price, mark price, funding fields, and\n" +
			"24h change data for that market.",
		Example: "# Show one state row for every market\n" +
			"  100x market state\n\n" +
			"# Show the detailed state record for BTCUSDT only\n" +
			"  100x market state BTCUSDT\n\n" +
			"# Extract symbol, last price, and next funding rate from all markets\n" +
			"  100x --json market state --jq 'map({symbol: .market, last, funding_rate_next})'",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Symbol = args[0]
			}
			return runState(cmd.Context(), opts)
		},
	}
	return c
}

func runState(ctx context.Context, opts *StateOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	if opts.Symbol == "" {
		resp, err := client.Market.MarketStateAll(ctx, futures.MarketStateAllReq{})
		if err != nil {
			return err
		}
		if resp == nil {
			resp = []futures.MarketStateItem{}
		}
		return f.IO.Render(resp, func() error { return printMarketStates(f.IO, resp) })
	}
	resp, err := client.Market.MarketState(ctx, futures.MarketStateReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error { return printMarketState(f.IO, *resp) })
}

func printMarketState(io *output.Renderer, s futures.MarketStateItem) error {
	return io.Object([]output.KV{
		{Key: "Symbol", Value: s.Market},
		{Key: "Last", Value: s.Last},
		{Key: "Change", Value: format.Percent(s.Change)},
		{Key: "High", Value: s.High},
		{Key: "Low", Value: s.Low},
		{Key: "Volume", Value: s.Volume},
		{Key: "Index", Value: s.IndexPrice},
		{Key: "Mark", Value: s.SignPrice},
		{Key: "Funding Next", Value: format.Percent(s.FundingRateNext)},
		{Key: "Funding Time", Value: fmt.Sprintf("%ds", s.FundingTime)},
	})
}

func printMarketStates(io *output.Renderer, rows []futures.MarketStateItem) error {
	out := make([][]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, []string{
			s.Market,
			s.Last,
			format.Percent(s.Change),
			s.Volume,
			s.IndexPrice,
			s.SignPrice,
			format.Percent(s.FundingRateNext),
		})
	}
	return io.Table([]output.Column{
		output.LCol("Symbol"),
		output.RCol("Last"), output.RCol("Change"), output.RCol("Volume"),
		output.RCol("Index"), output.RCol("Mark"), output.RCol("Funding Next"),
	}, out)
}
