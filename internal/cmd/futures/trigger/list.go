package trigger

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
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

// NewCmdTriggers builds the `futures triggers` shortcut for `futures trigger list`.
func NewCmdTriggers(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "triggers [symbol]",
		Short: "List active or finished triggers",
		Long:  "Shortcut for `100x futures trigger list`.",
		Example: "# List active BTCUSDT triggers\n" +
			"  100x futures triggers BTCUSDT\n\n" +
			"# List finished BTCUSDT triggers with page size 50\n" +
			"  100x futures triggers BTCUSDT --finished --page-size 50",
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
	c.Flags().BoolVar(&opts.Finished, "finished", false, "show finished triggers instead of active triggers")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "items per page")
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	if err := clierr.PositiveInt("--page", opts.Page); err != nil {
		return err
	}
	if err := clierr.PositiveInt("--page-size", opts.PageSize); err != nil {
		return err
	}
	if !opts.Finished {
		resp, err := f.Client.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{
			Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		records := stopRecords(resp.Records)
		return f.IO.Render(records, func() error { return printStops(f.IO, records, "No active triggers found.") })
	}
	resp, err := f.Client.Order.FinishedStopOrder(ctx, futures.FinishedStopOrderReq{
		Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := stopRecords(resp.Records)
	return f.IO.Render(records, func() error { return printStops(f.IO, records, "No finished triggers found.") })
}

func stopRecords(rows []futures.StopOrderItem) []futures.StopOrderItem {
	if rows == nil {
		return []futures.StopOrderItem{}
	}
	return rows
}

func printStops(io *output.Renderer, rows []futures.StopOrderItem, emptyMessage string) error {
	if len(rows) == 0 {
		return io.Emptyln(emptyMessage)
	}
	out := make([][]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, []string{
			s.ContractOrderID,
			s.Market,
			format.StopOrderType(io, s.ContractOrderType),
			format.Side(io, s.Side),
			format.StopOrderStatus(io, s.Status),
			s.TriggerPrice,
			s.OrderPrice,
			s.Size,
		})
	}
	return io.Table([]string{"Trigger ID", "Symbol", "Type", "Side", "Status", "Trigger Price", "Order Price", "Size"}, out)
}
