package trigger

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
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
		Use:   "list <symbol>",
		Short: "List active or finished triggers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().BoolVar(&opts.Finished, "finished", false, "list finished triggers")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 100, "page size")
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	if !opts.Finished {
		resp, err := f.Client.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{
			Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		records := stopRecords(resp.Records)
		return f.IO.Render(records, func() error { return printStops(f.IO, records) })
	}
	resp, err := f.Client.Order.FinishedStopOrder(ctx, futures.FinishedStopOrderReq{
		Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := stopRecords(resp.Records)
	return f.IO.Render(records, func() error { return printStops(f.IO, records) })
}

func stopRecords(rows []futures.StopOrderItem) []futures.StopOrderItem {
	if rows == nil {
		return []futures.StopOrderItem{}
	}
	return rows
}

func printStops(io *output.Renderer, rows []futures.StopOrderItem) error {
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
