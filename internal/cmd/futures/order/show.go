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
	"github.com/vika2603/100x-cli/internal/wire"
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
			"  100x futures order show BTCUSDT <order-id>\n\n" +
			"# Extract status, filled size, and SL/TP as JSON\n" +
			"  100x --json futures order show BTCUSDT <order-id> --jq '{status, filled, sl: .stop_loss_price, tp: .take_profit_price}'",
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
	opts.Symbol = wire.Market(opts.Symbol)
	f := opts.Factory
	if err := clierr.PositiveID("order-id", opts.OrderID); err != nil {
		return err
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	resp, err := client.Order.OrderDetail(ctx, futures.OrderDetailReq{
		Market: opts.Symbol, OrderID: opts.OrderID,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Position ID", Value: strconv.FormatInt(resp.PositionID, 10)},
			{Key: "Symbol", Value: resp.Market},
			{Key: "Side", Value: format.Side(f.IO, resp.Side)},
			{Key: "Type", Value: format.OrderType(resp.Type)},
			{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			{Key: "Mode", Value: format.PositionType(f.IO, resp.PositionType)},
			{Key: "Leverage", Value: format.EmptyDash(resp.Leverage) + "x"},
			{Key: "Price", Value: resp.Price},
			{Key: "Avg Price", Value: format.EmptyDash(resp.AvgPrice)},
			{Key: "Size", Value: resp.Volume},
			{Key: "Filled", Value: resp.Filled},
			{Key: "Left", Value: format.EmptyDash(resp.Left)},
			{Key: "Filled Value", Value: format.EmptyDash(resp.DealStock)},
			{Key: "Fee", Value: format.EmptyDash(resp.DealFee)},
			{Key: "SL", Value: format.EmptyDash(resp.StopLossPrice)},
			{Key: "TP", Value: format.EmptyDash(resp.TakeProfitPrice)},
			{Key: "Client ID", Value: resp.ClientOID},
			{Key: "Created", Value: format.UnixSecondsFloat(resp.CreateTime)},
			{Key: "Updated", Value: format.UnixSecondsFloat(resp.UpdateTime)},
		})
	})
}
