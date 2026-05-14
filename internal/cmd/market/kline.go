package market

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/timeexpr"
)

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
			"  100x market kline BTCUSDT --interval 1m --limit 20\n\n" +
			"# Query five-minute candles for the last hour using relative time expressions\n" +
			"  100x market kline BTCUSDT --since now-1h --until now --interval 5m\n\n" +
			"# Extract time, open, high, low, and close as JSON\n" +
			"  100x --json market kline BTCUSDT --interval 5m --limit 12 --jq 'map({time, open, high, low, close})'",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = format.Market(args[0])
			return runKline(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Interval, "interval", "1m", "Candle interval ("+strings.Join(complete.KlineIntervalAliases, ", ")+")")
	c.Flags().StringVar(&opts.Since, "since", "", "Start time: "+timeexpr.Help)
	c.Flags().StringVar(&opts.Until, "until", "", "End time: "+timeexpr.Help)
	c.Flags().IntVar(&opts.Limit, "limit", 20, "Latest candles to show")
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
	client, err := f.Futures()
	if err != nil {
		return err
	}
	resp, err := client.Market.MarketKline(ctx, futures.MarketKlineReq{
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
		return "", clierr.Usagef("unknown --interval %q (want %s)", s, strings.Join(complete.KlineIntervalAliases, ", "))
	}
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
	return io.Table([]output.Column{
		output.LCol("Time"),
		output.RCol("Open"), output.RCol("High"), output.RCol("Low"),
		output.RCol("Close"), output.RCol("Volume"),
	}, out)
}
