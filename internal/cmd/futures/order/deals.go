package order

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
)

// DealsOptions captures the flag-bound state of `order deals`.
type DealsOptions struct {
	Symbol    string
	StartTime int
	EndTime   int
	Page      int
	PageSize  int

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
	c.Flags().IntVar(&opts.StartTime, "since", 0, "start time (seconds)")
	c.Flags().IntVar(&opts.EndTime, "until", 0, "end time (seconds)")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "page size")
	return c
}

func runDeals(ctx context.Context, opts *DealsOptions) error {
	f := opts.Factory
	resp, err := f.Client.Order.OrderDeals(ctx, futures.OrderDealsReq{
		Market: opts.Symbol, StartTime: opts.StartTime, EndTime: opts.EndTime,
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
				strconv.Itoa(d.TradeID), d.Market, style.Side(f.IO, d.Side),
				d.Volume, d.Price, d.DealFee, d.DealProfit,
			})
		}
		return f.IO.Table([]string{"Trade ID", "Symbol", "Side", "Size", "Price", "Fee", "PnL"}, rows)
	})
}
