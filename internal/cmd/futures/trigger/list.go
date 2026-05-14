package trigger

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
	"github.com/vika2603/100x-cli/internal/wire"
)

// ListOptions captures the flag-bound state of `trigger list`.
type ListOptions struct {
	Symbol   string
	Finished bool
	Page     int
	PageSize int

	Factory *factory.Factory
}

// NewCmdList builds the `trigger list` cobra command.
func NewCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:     "list <symbol>",
		Aliases: []string{"ls"},
		Short:   "List active or finished triggers",
		Example: "# List active BTCUSDT triggers\n" +
			"  100x futures trigger list BTCUSDT\n\n" +
			"# List finished BTCUSDT triggers with page size 50\n" +
			"  100x futures trigger list BTCUSDT --finished --page-size 50\n\n" +
			"# Extract trigger id, type, side, trigger price, and status as JSON\n" +
			"  100x --json futures trigger list BTCUSDT --jq 'map({contract_order_id, contract_order_type, side, trigger_price, status})'",
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Symbol = args[0]
			}
			return runList(cmd.Context(), opts)
		},
	}
	addListFlags(c, opts)
	return c
}

func addListFlags(c *cobra.Command, opts *ListOptions) {
	c.Flags().BoolVar(&opts.Finished, "finished", false, "Show finished triggers instead of active triggers")
	c.Flags().IntVar(&opts.Page, "page", 1, "Page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "Items per page")
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
	if !opts.Finished {
		resp, err := client.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{
			Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		records := stopRecords(resp.Records)
		return f.IO.Render(records, func() error {
			return printStops(f.IO, records, "No active triggers found.", opts.Symbol != "")
		})
	}
	resp, err := client.Order.FinishedStopOrder(ctx, futures.FinishedStopOrderReq{
		Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := stopRecords(resp.Records)
	return f.IO.Render(records, func() error {
		return printStops(f.IO, records, "No finished triggers found.", opts.Symbol != "")
	})
}

func stopRecords(rows []futures.StopOrderItem) []futures.StopOrderItem {
	if rows == nil {
		return []futures.StopOrderItem{}
	}
	return rows
}

// printStops omits the Symbol column when symbolFiltered is true: the user
// already typed the symbol on the command line, so repeating it in every row
// is noise.
func printStops(io *output.Renderer, rows []futures.StopOrderItem, emptyMessage string, symbolFiltered bool) error {
	if len(rows) == 0 {
		return io.Emptyln(emptyMessage)
	}
	cols := []output.Column{output.LCol("Trigger ID")}
	if !symbolFiltered {
		cols = append(cols, output.LCol("Symbol"))
	}
	cols = append(cols,
		output.LCol("Type"), output.LCol("Side"), output.LCol("Status"),
		output.RCol("Trigger Price"), output.RCol("Order Price"), output.RCol("Size"),
		output.LCol("Order ID"), output.LCol("Position ID"), output.LCol("Created"),
	)
	out := make([][]string, 0, len(rows))
	for _, s := range rows {
		row := []string{s.ContractOrderID}
		if !symbolFiltered {
			row = append(row, s.Market)
		}
		row = append(row,
			format.StopOrderType(io, s.ContractOrderType),
			format.Side(io, s.Side),
			format.StopOrderStatus(io, s.Status),
			s.TriggerPrice,
			s.OrderPrice,
			s.Size,
			emptyDashID(s.OrderID),
			emptyDashID(s.PositionID),
			format.UnixAuto(s.OrderTime),
		)
		out = append(out, row)
	}
	return io.Table(cols, out)
}

func emptyDashID(id int64) string {
	if id == 0 {
		return "-"
	}
	return strconv.FormatInt(id, 10)
}
