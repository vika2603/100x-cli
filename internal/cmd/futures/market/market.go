// Package market wires the `100x futures market` verbs.
package market

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/timeexpr"
)

// NewCmdMarket returns the `market` group.
func NewCmdMarket(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "market",
		Short: "Public market data",
	}
	c.AddCommand(newCmdList(f), newCmdState(f), newCmdDepth(f), newCmdDeals(f), newCmdKline(f))
	return c
}

func newCmdList(f *factory.Factory) *cobra.Command {
	var includeUnavailable bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List all tradable instruments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := f.Client.Market.MarketList(cmd.Context(), futures.MarketListReq{})
			if err != nil {
				return err
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
		},
	}
	c.Flags().BoolVar(&includeUnavailable, "include-unavailable", false, "show unavailable markets too")
	return c
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
		Args:  cobra.MaximumNArgs(1),
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			resp, err := f.Client.Market.MarketDepth(cmd.Context(), futures.MarketDepthReq{Market: opts.Symbol, Merge: opts.TickSize})
			if err != nil {
				return err
			}
			trimDepth(resp, opts.Limit)
			return f.IO.Render(resp, func() error { return printDepth(f.IO, resp) })
		},
	}
	c.Flags().StringVar(&opts.TickSize, "tick-size", "", "price aggregation level")
	c.Flags().IntVar(&opts.Limit, "limit", 10, "levels per side")
	return c
}

func newCmdDeals(f *factory.Factory) *cobra.Command {
	var symbol string
	var limit int
	c := &cobra.Command{
		Use:   "deals <symbol>",
		Short: "List the latest public trades",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol = args[0]
			resp, err := f.Client.Market.MarketDeals(cmd.Context(), futures.MarketDealsReq{Market: symbol})
			if err != nil {
				return err
			}
			resp = limitSlice(resp, limit)
			return f.IO.Render(resp, func() error { return printMarketDeals(f.IO, resp) })
		},
	}
	c.Flags().IntVar(&limit, "limit", 20, "number of trades to show")
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runKline(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Interval, "interval", "1m", "candle interval")
	c.Flags().StringVar(&opts.Since, "since", "", "start time: "+timeexpr.Help)
	c.Flags().StringVar(&opts.Until, "until", "", "end time: "+timeexpr.Help)
	c.Flags().IntVar(&opts.Limit, "limit", 20, "number of latest candles to show")
	return c
}

func runKline(ctx context.Context, opts *KlineOptions) error {
	f := opts.Factory
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
		return "", fmt.Errorf("unknown --interval %q", s)
	}
}

func trimDepth(resp *futures.MarketDepthResp, limit int) {
	if limit <= 0 {
		return
	}
	resp.Asks = limitSlice(resp.Asks, limit)
	resp.Bids = limitSlice(resp.Bids, limit)
}

func printDepth(io *output.Renderer, d *futures.MarketDepthResp) error {
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
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func limitTail[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}
