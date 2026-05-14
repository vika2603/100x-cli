package order

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/cmd/futures/protection"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/wire"
)

// EditOptions captures the flag-bound state of `order edit`.
type EditOptions struct {
	Symbol  string
	OrderID string
	Price   string
	Size    string

	Factory *factory.Factory
}

// NewCmdEdit builds the `order edit` cobra command.
func NewCmdEdit(f *factory.Factory) *cobra.Command {
	opts := &EditOptions{Factory: f}
	c := &cobra.Command{
		Use:   "edit <symbol> <order-id>",
		Short: "Modify an open limit order",
		Long: "Modify an open limit order.\n\n" +
			"Edited limit orders are reissued with a new order id. Any attached SL/TP is re-attached\n" +
			"automatically onto the new id when possible.",
		Example: "# Rebook one BTCUSDT order to price 70500 and size 0.002\n" +
			"  100x futures order edit BTCUSDT <order-id> --price 70500 --size 0.002",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenOrderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Price, "price", "", "New limit price")
	c.Flags().StringVar(&opts.Size, "size", "", "New order quantity")
	_ = c.MarkFlagRequired("price")
	_ = c.MarkFlagRequired("size")
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	if opts.Price == "" || opts.Size == "" {
		return clierr.Usagef("order edit: --price and --size are required")
	}
	if err := clierr.PositiveID("order-id", opts.OrderID); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--price", opts.Price); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	old := protection.Order{Symbol: opts.Symbol, OrderID: opts.OrderID}
	oldState, err := old.Inspect(ctx, client)
	if err != nil {
		return err
	}
	if oldState.HasAny() && oldState.CrossOrderConflict {
		return fmt.Errorf("order edit would need to reattach SL/TP for %s, but another order on the same position has active triggers; edit/cancel those triggers first", opts.OrderID)
	}
	resp, err := client.Order.EditLimitOrder(ctx, futures.LimitOrderEditReq{
		Market:   opts.Symbol,
		OrderID:  opts.OrderID,
		Price:    opts.Price,
		Quantity: opts.Size,
	})
	if err != nil {
		return err
	}
	if oldState.HasAny() {
		newOrderID := strconv.FormatInt(resp.OrderID, 10)
		fresh := protection.Order{Symbol: opts.Symbol, OrderID: newOrderID}
		want := dropTriggerIDs(oldState)
		if err := fresh.Apply(ctx, client, protection.State{}, want); err != nil {
			return fmt.Errorf("order edited to %d but failed to reattach SL/TP: %w", resp.OrderID, err)
		}
		if err := fresh.Verify(ctx, client, want); err != nil {
			return fmt.Errorf("order edited to %d but failed to verify reattached SL/TP: %w", resp.OrderID, err)
		}
		if refreshed, err := client.Order.OrderDetail(ctx, futures.OrderDetailReq{
			Market: opts.Symbol, OrderID: newOrderID,
		}); err == nil {
			resp.OrderItem = *refreshed
		} else {
			resp.StopLossPrice = want.SL.Price
			resp.TakeProfitPrice = want.TP.Price
		}
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
			{Key: "SL", Value: format.EmptyDash(resp.StopLossPrice)},
			{Key: "TP", Value: format.EmptyDash(resp.TakeProfitPrice)},
		})
	})
}

// dropTriggerIDs strips the standalone TriggerIDs from a State so it can be
// re-applied to a freshly rebooked order. The old triggers were canceled
// alongside the old order; keeping the IDs would route Apply into the
// edit-existing-trigger branch with stale identifiers.
func dropTriggerIDs(s protection.State) protection.State {
	s.SL.TriggerID = ""
	s.TP.TriggerID = ""
	s.CrossOrderConflict = false
	return s
}
