package order

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/timeexpr"
)

// DealsOptions captures the flag-bound state of `order deals`.
type DealsOptions struct {
	Symbol   string
	Since    string
	Until    string
	Page     int
	PageSize int

	Factory *factory.Factory
}

// NewCmdDeals builds the `order deals` cobra command.
func NewCmdDeals(f *factory.Factory) *cobra.Command {
	opts := &DealsOptions{Factory: f, Page: 1, PageSize: 20}
	c := &cobra.Command{
		Use:   "deals",
		Short: "List trade-level fill records",
		Long: "List trade-level fill records.\n\n" +
			"When --since is set and --until is omitted, the CLI uses the current time as the end of the window.",
		Example: "# List recent private fills for BTCUSDT with page size 20\n" +
			"  100x futures order deals --symbol BTCUSDT --page-size 20\n\n" +
			"# List private BTCUSDT fills from the last 24 hours\n" +
			"  100x futures order deals --symbol BTCUSDT --since now-24h\n\n" +
			"# Extract trade id, order id, side, price, size, and pnl as JSON\n" +
			"  100x --json futures order deals --symbol BTCUSDT --jq 'map({trade_id, order_id, side, price, volume, deal_profit})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDeals(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "only show fills for this symbol")
	c.Flags().StringVar(&opts.Since, "since", "", "start time: "+timeexpr.Help)
	c.Flags().StringVar(&opts.Until, "until", "", "end time: "+timeexpr.Help)
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "items per page")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	_ = c.RegisterFlagCompletionFunc("since", complete.TimeExpressions)
	_ = c.RegisterFlagCompletionFunc("until", complete.TimeExpressions)
	return c
}

func runDeals(ctx context.Context, opts *DealsOptions) error {
	f := opts.Factory
	if err := clierr.PositiveInt("--page", opts.Page); err != nil {
		return err
	}
	if err := clierr.PositiveInt("--page-size", opts.PageSize); err != nil {
		return err
	}
	startTime, endTime, err := timeexpr.ResolveRange(opts.Since, opts.Until)
	if err != nil {
		return err
	}
	resp, err := f.Client.Order.OrderDeals(ctx, futures.OrderDealsReq{
		Market: opts.Symbol, StartTime: startTime, EndTime: endTime,
		Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := resp.Records
	if records == nil {
		records = []futures.OrderDealItem{}
	}
	return f.IO.Render(records, func() error {
		if len(records) == 0 {
			return f.IO.Emptyln("No fills found.")
		}
		rows := make([][]string, 0, len(records))
		for _, d := range records {
			rows = append(rows, []string{
				strconv.Itoa(d.TradeID), d.Market, format.Side(f.IO, d.Side),
				d.Volume, d.Price, d.DealFee, d.DealProfit, format.UnixSecondsFloat(float64(d.Time)),
			})
		}
		return f.IO.Table([]string{"Trade ID", "Symbol", "Side", "Size", "Price", "Fee", "PnL", "Time"}, rows)
	})
}
