package order

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

// ListOptions captures the flag-bound state of `order list`.
type ListOptions struct {
	Market   string
	Status   string
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
		resp, err := f.Client.Order.PendingOrder(ctx, futures.PendingOrderReq{
			Market: opts.Market, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error { return printOrders(f.IO, resp.Records) })
	case "closed":
		resp, err := f.Client.Order.FinishedOrder(ctx, futures.FinishedOrderReq{
			Market: opts.Market, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error { return printOrders(f.IO, resp.Records) })
	}
	return fmt.Errorf("unknown --status %q (want open|closed)", opts.Status)
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
			o.ClientOID,
		})
	}
	return io.Table([]string{"ID", "Market", "Side", "Status", "Price", "Qty", "Filled", "Client Order ID"}, out)
}
