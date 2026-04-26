package trigger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// ListOptions captures the flag-bound state of `trigger list`.
type ListOptions struct {
	Market   string
	Status   string
	Page     int
	PageSize int

	Factory *factory.Factory
}

// NewCmdList builds the `trigger list` cobra command.
func NewCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "list",
		Short: "List active or finished triggers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "filter by market")
	c.Flags().StringVar(&opts.Status, "status", "open", "open | closed")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 50, "page size")
	_ = c.RegisterFlagCompletionFunc("status", cobra.FixedCompletions([]string{"open", "closed"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	switch opts.Status {
	case "open", "":
		resp, err := f.Client.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{
			Market: opts.Market, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error { return printStops(f.IO, resp.Records) })
	case "closed":
		resp, err := f.Client.Order.FinishedStopOrder(ctx, futures.FinishedStopOrderReq{
			Market: opts.Market, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error { return printStops(f.IO, resp.Records) })
	}
	return fmt.Errorf("unknown --status %q (want open|closed)", opts.Status)
}

func printStops(io *output.Renderer, rows []futures.StopOrderItem) error {
	out := make([][]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, []string{
			strconv.FormatInt(s.OrderID, 10),
			s.Market,
			style.StopOrderType(io, s.ContractOrderType),
			style.Side(io, s.Side),
			style.StopOrderStatus(io, s.Status),
			s.TriggerPrice,
			s.OrderPrice,
			s.Size,
		})
	}
	return io.Table([]string{"ID", "Market", "Type", "Side", "Status", "Trigger Price", "Order Price", "Size"}, out)
}
