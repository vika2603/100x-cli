// Package market wires the `100x futures market` verbs.
package market

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdMarket returns the `market` group.
func NewCmdMarket(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "market",
		Short: "Public market data",
	}
	c.AddCommand(newCmdSymbols(f), newCmdTicker(f), newCmdDepth(f), newCmdTrades(f), newCmdKline(f))
	return c
}

func newCmdSymbols(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "symbols",
		Short: "List all tradable instruments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := f.Client.Market.MarketList(cmd.Context(), futures.MarketListReq{})
			if err != nil {
				return err
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
}

// TickerOptions for `market ticker`.
type TickerOptions struct {
	Market string
	All    bool

	Factory *factory.Factory
}

func newCmdTicker(f *factory.Factory) *cobra.Command {
	opts := &TickerOptions{Factory: f}
	c := &cobra.Command{
		Use:   "ticker",
		Short: "Show one or all market tickers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTicker(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().BoolVar(&opts.All, "all", false, "fetch every market")
	return c
}

func runTicker(ctx context.Context, opts *TickerOptions) error {
	f := opts.Factory
	if opts.All {
		resp, err := f.Client.Market.MarketStateAll(ctx, futures.MarketStateAllReq{})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, nil)
	}
	resp, err := f.Client.Market.MarketState(ctx, futures.MarketStateReq{Market: opts.Market})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, nil)
}

// DepthOptions for `market depth`.
type DepthOptions struct {
	Market string
	Merge  string

	Factory *factory.Factory
}

func newCmdDepth(f *factory.Factory) *cobra.Command {
	opts := &DepthOptions{Factory: f}
	c := &cobra.Command{
		Use:   "depth",
		Short: "Show the order-book snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := f.Client.Market.MarketDepth(cmd.Context(), futures.MarketDepthReq{Market: opts.Market, Merge: opts.Merge})
			if err != nil {
				return err
			}
			return f.IO.Render(resp, nil)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Merge, "merge", "", "price-step merge (e.g. 0.1)")
	_ = c.MarkFlagRequired("market")
	return c
}

func newCmdTrades(f *factory.Factory) *cobra.Command {
	var market string
	c := &cobra.Command{
		Use:   "trades",
		Short: "List the latest public trades",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := f.Client.Market.MarketDeals(cmd.Context(), futures.MarketDealsReq{Market: market})
			if err != nil {
				return err
			}
			return f.IO.Render(resp, nil)
		},
	}
	c.Flags().StringVar(&market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

// KlineOptions for `market kline`.
type KlineOptions struct {
	Market    string
	Type      string
	StartTime int
	EndTime   int

	Factory *factory.Factory
}

func newCmdKline(f *factory.Factory) *cobra.Command {
	opts := &KlineOptions{Factory: f}
	c := &cobra.Command{
		Use:   "kline",
		Short: "Get candlestick history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runKline(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Type, "type", "1m", "candle type (1m, 5m, 1h, ...)")
	c.Flags().IntVar(&opts.StartTime, "start", 0, "start time (seconds)")
	c.Flags().IntVar(&opts.EndTime, "end", 0, "end time (seconds)")
	_ = c.MarkFlagRequired("market")
	return c
}

func runKline(ctx context.Context, opts *KlineOptions) error {
	f := opts.Factory
	resp, err := f.Client.Market.MarketKline(ctx, futures.MarketKlineReq{
		Market: opts.Market, Type: opts.Type, StartTime: opts.StartTime, EndTime: opts.EndTime,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, nil)
}
