// Package market wires the `100x futures market` verbs.
package market

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/timeexpr"
)

// NewCmdMarket returns the `market` group.
func NewCmdMarket(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "market",
		Aliases: []string{"m"},
		Short:   "Public market data",
		Long: "Read public futures market data.\n\n" +
			"These commands do not place trades or require private credentials. Use them to discover\n" +
			"symbols, inspect ticker state, read the current order book, list recent public trades,\n" +
			"or query candle history for charting and diagnostics.",
		Example: "# Show the latest ticker snapshot for BTCUSDT\n" +
			"  100x futures market state BTCUSDT\n\n" +
			"# Show the top 5 bid and ask levels for BTCUSDT\n" +
			"  100x futures market depth BTCUSDT --limit 5",
	}
	c.AddCommand(newCmdList(f), newCmdState(f), newCmdDepth(f), newCmdDeals(f), newCmdKline(f))
	return c
}

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
		return f.IO.Table([]string{"Symbol", "Base", "Quote", "Tick Size", "Maker Fee", "Taker Fee"}, rows)
	})
}

// StateOptions for `market state`.
type StateOptions struct {
	Symbol string

	Factory *factory.Factory
}

func newCmdState(f *factory.Factory) *cobra.Command {
	opts := &StateOptions{Factory: f}
	c := &cobra.Command{
		Use:   "state [symbol]",
		Short: "Show one or all market states",
		Long: "Show ticker-style market state.\n\n" +
			"Without a symbol, the command prints a table for every market. With a symbol, it prints\n" +
			"one detailed record including last price, index price, mark price, funding fields, and\n" +
			"24h change data for that market.",
		Example: "# Show one state row for every market\n" +
			"  100x futures market state\n\n" +
			"# Show the detailed state record for BTCUSDT only\n" +
			"  100x futures market state BTCUSDT\n\n" +
			"# Extract symbol, last price, and next funding rate from all markets\n" +
			"  100x --json futures market state --jq 'map({symbol: .market, last, funding_rate_next})'",
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
	f := opts.Factory
	if opts.Symbol == "" {
		resp, err := f.Client.Market.MarketStateAll(ctx, futures.MarketStateAllReq{})
		if err != nil {
			return err
		}
		if resp == nil {
			resp = []futures.MarketStateItem{}
		}
		return f.IO.Render(resp, func() error { return printMarketStates(f.IO, resp) })
	}
	resp, err := f.Client.Market.MarketState(ctx, futures.MarketStateReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error { return printMarketState(f.IO, *resp) })
}

func printMarketState(io *output.Renderer, s futures.MarketStateItem) error {
	return io.Object([]output.KV{
		{Key: "Symbol", Value: s.Market},
		{Key: "Last", Value: s.Last},
		{Key: "Change", Value: s.Change},
		{Key: "High", Value: s.High},
		{Key: "Low", Value: s.Low},
		{Key: "Volume", Value: s.Volume},
		{Key: "Index", Value: s.IndexPrice},
		{Key: "Mark", Value: s.SignPrice},
		{Key: "Funding Next", Value: s.FundingRateNext},
		{Key: "Funding Time", Value: strconv.FormatInt(s.FundingTime, 10)},
	})
}

func printMarketStates(io *output.Renderer, rows []futures.MarketStateItem) error {
	out := make([][]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, []string{
			s.Market,
			s.Last,
			s.Change,
			s.Volume,
			s.IndexPrice,
			s.SignPrice,
			s.FundingRateNext,
		})
	}
	return io.Table([]string{"Symbol", "Last", "Change", "Volume", "Index", "Mark", "Funding Next"}, out)
}

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
			"merged book levels when you want a less granular view of the market depth.",
		Example: "# Show the current order book for BTCUSDT\n" +
			"  100x futures market depth BTCUSDT\n\n" +
			"# Merge levels by tick size 0.1 and show 20 bids and 20 asks\n" +
			"  100x futures market depth BTCUSDT --tick-size 0.1 --limit 20",
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
	c.Flags().StringVar(&opts.TickSize, "tick-size", "", "merge book levels by this tick size")
	c.Flags().IntVar(&opts.Limit, "limit", 10, "levels to show on each side")
	return c
}

func newCmdDeals(f *factory.Factory) *cobra.Command {
	var symbol string
	var limit int
	c := &cobra.Command{
		Use:   "deals <symbol>",
		Short: "List the latest public trades",
		Long: "List the latest public trades for one symbol.\n\n" +
			"This is market-wide trade flow, not your private fills. Use `order deals` when you want\n" +
			"your account's fill history instead of the public tape.",
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
	c.Flags().IntVar(&limit, "limit", 20, "recent trades to show")
	return c
}

// KlineOptions for `market kline`.
type KlineOptions struct {
	Symbol   string
	Interval string
	Since    string
	Until    string
	Limit    int

	Factory *factory.Factory
}

func newCmdKline(f *factory.Factory) *cobra.Command {
	opts := &KlineOptions{Factory: f}
	c := &cobra.Command{
		Use:   "kline <symbol>",
		Short: "Get candlestick history",
		Long: "Get candlestick history.\n\n" +
			"When --since is set and --until is omitted, the CLI uses the current time as the end of the window.",
		Example: "# Show the latest 20 one-minute candles for BTCUSDT\n" +
			"  100x futures market kline BTCUSDT --interval 1m --limit 20\n\n" +
			"# Query five-minute candles for the last hour using relative time expressions\n" +
			"  100x futures market kline BTCUSDT --since now-1h --until now --interval 5m\n\n" +
			"# Extract time, open, high, low, and close as JSON\n" +
			"  100x --json futures market kline BTCUSDT --interval 5m --limit 12 --jq 'map({time, open, high, low, close})'",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runKline(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Interval, "interval", "1m", "candle interval, for example 1m or 5m")
	c.Flags().StringVar(&opts.Since, "since", "", "start time: "+timeexpr.Help)
	c.Flags().StringVar(&opts.Until, "until", "", "end time: "+timeexpr.Help)
	c.Flags().IntVar(&opts.Limit, "limit", 20, "latest candles to show")
	_ = c.RegisterFlagCompletionFunc("interval", complete.KlineIntervals)
	_ = c.RegisterFlagCompletionFunc("since", complete.TimeExpressions)
	_ = c.RegisterFlagCompletionFunc("until", complete.TimeExpressions)
	return c
}

func runKline(ctx context.Context, opts *KlineOptions) error {
	f := opts.Factory
	if err := clierr.PositiveInt("--limit", opts.Limit); err != nil {
		return err
	}
	interval, err := parseInterval(opts.Interval)
	if err != nil {
		return err
	}
	startTime, endTime, err := timeexpr.ResolveRange(opts.Since, opts.Until)
	if err != nil {
		return err
	}
	resp, err := f.Client.Market.MarketKline(ctx, futures.MarketKlineReq{
		Market: opts.Symbol, Type: interval, StartTime: startTime, EndTime: endTime,
	})
	if err != nil {
		return err
	}
	if resp == nil {
		resp = []futures.MarketKlineItem{}
	}
	resp = limitTail(resp, opts.Limit)
	return f.IO.Render(resp, func() error { return printKlines(f.IO, resp) })
}

func parseInterval(s string) (string, error) {
	switch s {
	case "1m":
		return "1min", nil
	case "5m":
		return "5min", nil
	case "10m":
		return "10min", nil
	case "15m":
		return "15min", nil
	case "30m":
		return "30min", nil
	case "1h":
		return "1hour", nil
	case "2h":
		return "2hour", nil
	case "4h":
		return "4hour", nil
	case "6h":
		return "6hour", nil
	case "12h":
		return "12hour", nil
	case "1d":
		return "1day", nil
	case "1w":
		return "1week", nil
	case "1M":
		return "1month", nil
	case "1min", "5min", "10min", "15min", "30min",
		"1hour", "2hour", "4hour", "6hour", "12hour",
		"1day", "1week", "1month":
		return s, nil
	default:
		return "", clierr.Usagef("unknown --interval %q", s)
	}
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
	return io.Table([]string{"Side", "Price", "Size"}, rows)
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
	return io.Table([]string{"Trade ID", "Side", "Price", "Size", "Time"}, out)
}

func printKlines(io *output.Renderer, rows []futures.MarketKlineItem) error {
	if len(rows) == 0 {
		return io.Emptyln("No candles found.")
	}
	out := make([][]string, 0, len(rows))
	for _, k := range rows {
		out = append(out, []string{
			format.UnixMillis(k.Time),
			k.Open,
			k.High,
			k.Low,
			k.Close,
			k.Volume,
		})
	}
	return io.Table([]string{"Time", "Open", "High", "Low", "Close", "Volume"}, out)
}

func limitSlice[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func limitTail[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}
