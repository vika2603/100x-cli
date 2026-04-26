// Package market wires the `100x futures market` verbs.
package market

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/output"
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
			return f.IO.Render(resp, nil)
		},
	}
	c.Flags().StringVar(&opts.TickSize, "tick-size", "", "price aggregation level")
	return c
}

func newCmdDeals(f *factory.Factory) *cobra.Command {
	var symbol string
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
			return f.IO.Render(resp, nil)
		},
	}
	return c
}

// KlineOptions for `market kline`.
type KlineOptions struct {
	Symbol    string
	Interval  string
	StartTime int
	EndTime   int

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
	c.Flags().IntVar(&opts.StartTime, "since", 0, "start time (seconds)")
	c.Flags().IntVar(&opts.EndTime, "until", 0, "end time (seconds)")
	return c
}

func runKline(ctx context.Context, opts *KlineOptions) error {
	f := opts.Factory
	interval, err := parseInterval(opts.Interval)
	if err != nil {
		return err
	}
	resp, err := f.Client.Market.MarketKline(ctx, futures.MarketKlineReq{
		Market: opts.Symbol, Type: interval, StartTime: opts.StartTime, EndTime: opts.EndTime,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, nil)
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
