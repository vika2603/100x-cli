package order

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
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
	opts := &DealsOptions{Factory: f, Page: 1, PageSize: 100}
	c := &cobra.Command{
		Use:   "deals",
		Short: "List trade-level fill records",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDeals(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "filter by symbol")
	c.Flags().StringVar(&opts.Since, "since", "", "start time: "+timeexpr.Help)
	c.Flags().StringVar(&opts.Until, "until", "", "end time: "+timeexpr.Help)
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "page size")
	return c
}

func runDeals(ctx context.Context, opts *DealsOptions) error {
	f := opts.Factory
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
