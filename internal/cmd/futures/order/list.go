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
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/timeexpr"
	"github.com/vika2603/100x-cli/internal/wire"
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
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List open or finished orders",
		Long: "List open or finished orders.\n\n" +
			"When --since is set and --until is omitted, the CLI uses the current time as the end of the window.",
		Example: "# List every open order in the account\n" +
			"  100x futures order list\n\n" +
			"# List open orders for BTCUSDT only\n" +
			"  100x futures order list --symbol BTCUSDT\n\n" +
			"# List finished BTCUSDT orders from the last 24 hours with page size 50\n" +
			"  100x futures order list --finished --symbol BTCUSDT --since now-24h --page-size 50\n\n" +
			"# Extract order id, side, price, size, and status as JSON\n" +
			"  100x --json futures order list --symbol BTCUSDT --jq 'map({order_id, side, price, volume, status})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	addListFlags(c, opts)
	return c
}

func addListFlags(c *cobra.Command, opts *ListOptions) {
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "Only show orders for this symbol")
	c.Flags().BoolVar(&opts.Finished, "finished", false, "Show finished orders instead of open orders")
	c.Flags().StringVar(&opts.Since, "since", "", "Start time: "+timeexpr.Help+" (finished only)")
	c.Flags().StringVar(&opts.Until, "until", "", "End time: "+timeexpr.Help+" (finished only)")
	c.Flags().IntVar(&opts.Page, "page", 1, "Page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "Items per page")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	_ = c.RegisterFlagCompletionFunc("since", complete.TimeExpressions)
	_ = c.RegisterFlagCompletionFunc("until", complete.TimeExpressions)
}

func runList(ctx context.Context, opts *ListOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	f := opts.Factory
	if err := clierr.PositiveInt("--page", opts.Page); err != nil {
		return err
	}
	if err := clierr.PositiveInt("--page-size", opts.PageSize); err != nil {
		return err
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	symbolFiltered := opts.Symbol != ""
	if !opts.Finished {
		resp, err := client.Order.PendingOrder(ctx, futures.PendingOrderReq{
			Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		records := orderRecords(resp.Records)
		return f.IO.Render(records, func() error {
			// Pending orders are always limit (market orders fill or reject
			// immediately and never reach this list), so the Type column would
			// be a constant LIMIT and is omitted.
			return printOrders(f.IO, records, "Created", "No open orders found.", false, symbolFiltered, func(o futures.OrderItem) string {
				return format.UnixSecondsFloat(o.CreateTime)
			})
		})
	}
	startTime, endTime, err := timeexpr.ResolveRange(opts.Since, opts.Until)
	if err != nil {
		return err
	}
	resp, err := client.Order.FinishedOrder(ctx, futures.FinishedOrderReq{
		Market: opts.Symbol, StartTime: startTime, EndTime: endTime,
		Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := orderRecords(resp.Records)
	return f.IO.Render(records, func() error {
		return printOrders(f.IO, records, "Finished", "No finished orders found.", true, symbolFiltered, func(o futures.OrderItem) string {
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

func printOrders(io *output.Renderer, rows []futures.OrderItem, timeHeader, emptyMessage string, showType, symbolFiltered bool, timeValue func(futures.OrderItem) string) error {
	if len(rows) == 0 {
		return io.Emptyln(emptyMessage)
	}
	cols := []output.Column{output.LCol("Order ID")}
	if !symbolFiltered {
		cols = append(cols, output.LCol("Symbol"))
	}
	cols = append(cols, output.LCol("Side"))
	if showType {
		cols = append(cols, output.LCol("Type"))
	}
	cols = append(cols,
		output.LCol("Status"),
		output.RCol("Price"), output.RCol("Size"), output.RCol("Filled"),
		output.RCol("SL"), output.RCol("TP"),
		output.LCol("Client ID"), output.LCol(timeHeader),
	)
	out := make([][]string, 0, len(rows))
	for _, o := range rows {
		row := []string{strconv.FormatInt(o.OrderID, 10)}
		if !symbolFiltered {
			row = append(row, o.Market)
		}
		row = append(row, format.Side(io, o.Side))
		if showType {
			row = append(row, format.OrderType(o.Type))
		}
		row = append(row,
			format.OrderStatus(io, o.Status),
			o.Price,
			o.Volume,
			o.Filled,
			format.EmptyDash(o.StopLossPrice),
			format.EmptyDash(o.TakeProfitPrice),
			o.ClientOID,
			timeValue(o),
		)
		out = append(out, row)
	}
	return io.Table(cols, out)
}
