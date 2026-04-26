package order

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// ListOptions captures the flag-bound state of `order list`.
type ListOptions struct {
	Symbol    string
	Finished  bool
	StartTime int
	EndTime   int
	Page      int
	PageSize  int

	Factory *factory.Factory
}

// NewCmdList builds the `order list` cobra command.
func NewCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "list",
		Short: "List open or finished orders",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "filter by symbol")
	c.Flags().BoolVar(&opts.Finished, "finished", false, "list finished orders instead of pending")
	c.Flags().IntVar(&opts.StartTime, "since", 0, "start time (seconds, finished orders only)")
	c.Flags().IntVar(&opts.EndTime, "until", 0, "end time (seconds, finished orders only)")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 100, "page size")
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	if !opts.Finished {
		resp, err := f.Client.Order.PendingOrder(ctx, futures.PendingOrderReq{
			Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		records := orderRecords(resp.Records)
		return f.IO.Render(records, func() error { return printOrders(f.IO, records) })
	}
	resp, err := f.Client.Order.FinishedOrder(ctx, futures.FinishedOrderReq{
		Market: opts.Symbol, StartTime: opts.StartTime, EndTime: opts.EndTime,
		Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := orderRecords(resp.Records)
	return f.IO.Render(records, func() error { return printOrders(f.IO, records) })
}

func orderRecords(rows []futures.OrderItem) []futures.OrderItem {
	if rows == nil {
		return []futures.OrderItem{}
	}
	return rows
}

func printOrders(io *output.Renderer, rows []futures.OrderItem) error {
	out := make([][]string, 0, len(rows))
	for _, o := range rows {
		out = append(out, []string{
			strconv.FormatInt(o.OrderID, 10),
			o.Market,
			style.Side(io, o.Side),
			style.OrderStatus(io, o.Status),
			o.Price,
			o.Volume,
			o.Filled,
			emptyDash(o.StopLossPrice),
			emptyDash(o.TakeProfitPrice),
			o.ClientOID,
		})
	}
	return io.Table([]string{"Order ID", "Symbol", "Side", "Status", "Price", "Size", "Filled", "SL", "TP", "Client ID"}, out)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
