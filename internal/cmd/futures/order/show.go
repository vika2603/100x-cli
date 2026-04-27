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
)

// ShowOptions captures the flag-bound state of `order show`.
type ShowOptions struct {
	Symbol  string
	OrderID string

	Factory *factory.Factory
}

// NewCmdShow builds the `order show` cobra command.
func NewCmdShow(f *factory.Factory) *cobra.Command {
	opts := &ShowOptions{Factory: f}
	c := &cobra.Command{
		Use:   "show <symbol> <order-id>",
		Short: "Show one order's full record",
		Example: "# Show one BTCUSDT order with status, SL/TP, client id, and timestamps\n" +
			"  100x futures order show BTCUSDT <order-id>",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenOrderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runShow(cmd.Context(), opts)
		},
	}
	return c
}

func runShow(ctx context.Context, opts *ShowOptions) error {
	f := opts.Factory
	if err := clierr.PositiveID("order-id", opts.OrderID); err != nil {
		return err
	}
	resp, err := f.Client.Order.OrderDetail(ctx, futures.OrderDetailReq{
		Market: opts.Symbol, OrderID: opts.OrderID,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Symbol", Value: resp.Market},
			{Key: "Side", Value: format.Side(f.IO, resp.Side)},
			{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			{Key: "Price", Value: resp.Price},
			{Key: "Size", Value: resp.Volume},
			{Key: "Filled", Value: resp.Filled},
			{Key: "SL", Value: emptyDash(resp.StopLossPrice)},
			{Key: "TP", Value: emptyDash(resp.TakeProfitPrice)},
			{Key: "Client ID", Value: resp.ClientOID},
			{Key: "Created", Value: format.UnixSecondsFloat(resp.CreateTime)},
			{Key: "Updated", Value: format.UnixSecondsFloat(resp.UpdateTime)},
		})
	})
}
