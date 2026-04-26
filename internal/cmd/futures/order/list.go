package order

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/timeexpr"
)

// ListOptions captures the flag-bound state of `order list`.
type ListOptions struct {
	Symbol   string
	Finished bool
	Since    string
	Until    string
	Page     int
	PageSize int

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
	c.Flags().StringVar(&opts.Since, "since", "", "start time: "+timeexpr.Help+" (finished only)")
	c.Flags().StringVar(&opts.Until, "until", "", "end time: "+timeexpr.Help+" (finished only)")
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
		return f.IO.Render(records, func() error {
			return printOrders(f.IO, records, "Created", func(o futures.OrderItem) string {
				return format.UnixSecondsFloat(o.CreateTime)
			})
		})
	}
	startTime, endTime, err := timeexpr.ResolveRange(opts.Since, opts.Until)
	if err != nil {
		return err
	}
	resp, err := f.Client.Order.FinishedOrder(ctx, futures.FinishedOrderReq{
		Market: opts.Symbol, StartTime: startTime, EndTime: endTime,
		Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := orderRecords(resp.Records)
	return f.IO.Render(records, func() error {
		return printOrders(f.IO, records, "Updated", func(o futures.OrderItem) string {
			return format.UnixSecondsFloat(o.UpdateTime)
		})
	})
}

func orderRecords(rows []futures.OrderItem) []futures.OrderItem {
	if rows == nil {
		return []futures.OrderItem{}
	}
	return rows
}

func printOrders(io *output.Renderer, rows []futures.OrderItem, timeHeader string, timeValue func(futures.OrderItem) string) error {
	out := make([][]string, 0, len(rows))
	for _, o := range rows {
		out = append(out, []string{
			strconv.FormatInt(o.OrderID, 10),
			o.Market,
			format.Side(io, o.Side),
			format.OrderStatus(io, o.Status),
			o.Price,
			o.Volume,
			o.Filled,
			emptyDash(o.StopLossPrice),
			emptyDash(o.TakeProfitPrice),
			o.ClientOID,
			timeValue(o),
		})
	}
	return io.Table([]string{"Order ID", "Symbol", "Side", "Status", "Price", "Size", "Filled", "SL", "TP", "Client ID", timeHeader}, out)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
