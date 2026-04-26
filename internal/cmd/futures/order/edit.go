package order

import (
	"context"
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// EditOptions captures the flag-bound state of `order edit`.
type EditOptions struct {
	Market   string
	OrderID  string
	Price    string
	Quantity string

	Factory *factory.Factory
}

// NewCmdEdit builds the `order edit` cobra command.
func NewCmdEdit(f *factory.Factory) *cobra.Command {
	opts := &EditOptions{Factory: f}
	c := &cobra.Command{
		Use:   "edit <order-id>",
		Short: "Modify an open limit order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderID = args[0]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Price, "price", "", "new price")
	c.Flags().StringVar(&opts.Quantity, "qty", "", "new quantity")
	_ = c.MarkFlagRequired("market")
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	if opts.Price == "" && opts.Quantity == "" {
		return errors.New("order edit: at least one of --price or --qty is required")
	}
	f := opts.Factory
	resp, err := f.Client.Order.EditLimitOrder(ctx, futures.LimitOrderEditReq{
		Market:   opts.Market,
		OrderID:  opts.OrderID,
		Price:    opts.Price,
		Quantity: opts.Quantity,
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
		})
	})
}
