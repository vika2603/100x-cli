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

// ShowOptions captures the flag-bound state of `order show`.
type ShowOptions struct {
	Market  string
	OrderID string

	Factory *factory.Factory
}

// NewCmdShow builds the `order show` cobra command.
func NewCmdShow(f *factory.Factory) *cobra.Command {
	opts := &ShowOptions{Factory: f}
	c := &cobra.Command{
		Use:   "show <order-id>",
		Short: "Show one order's full record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderID = args[0]
			return runShow(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

func runShow(ctx context.Context, opts *ShowOptions) error {
	f := opts.Factory
	resp, err := f.Client.Order.OrderDetail(ctx, futures.OrderDetailReq{
		Market: opts.Market, OrderID: opts.OrderID,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Market", Value: resp.Market},
			{Key: "Side", Value: style.Side(f.IO, resp.Side)},
			{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
			{Key: "Price", Value: resp.Price},
			{Key: "Qty", Value: resp.Volume},
			{Key: "Filled", Value: resp.Filled},
			{Key: "Client Order ID", Value: resp.ClientOID},
		})
	})
}
