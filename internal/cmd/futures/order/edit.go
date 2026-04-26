package order

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
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
			"The gateway rebooks edited limit orders as a new order id. The CLI preserves attached SL/TP only when the backend can do so safely.",
		Example: "# Rebook one BTCUSDT order to price 70500 and size 0.002\n" +
			"  100x futures order edit BTCUSDT <order-id> --price 70500 --size 0.002",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Price, "price", "", "new limit price")
	c.Flags().StringVar(&opts.Size, "size", "", "new order quantity")
	_ = c.MarkFlagRequired("price")
	_ = c.MarkFlagRequired("size")
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	if opts.Price == "" || opts.Size == "" {
		return fmt.Errorf("order edit: --price and --size are required")
	}
	f := opts.Factory
	protection, err := readOrderProtection(ctx, f.Client, opts.Symbol, opts.OrderID)
	if err != nil {
		return err
	}
	if protection.HasAny() && protection.ConflictsWithOtherOrder {
		return fmt.Errorf("order edit would need to reattach SL/TP for %s, but another order on the same position has active triggers; edit/cancel those triggers first", opts.OrderID)
	}
	resp, err := f.Client.Order.EditLimitOrder(ctx, futures.LimitOrderEditReq{
		Market:   opts.Symbol,
		OrderID:  opts.OrderID,
		Price:    opts.Price,
		Quantity: opts.Size,
	})
	if err != nil {
		return err
	}
	if protection.HasAny() {
		newOrderID := strconv.FormatInt(resp.OrderID, 10)
		if _, err := f.Client.Order.StopOrderClose(ctx, futures.StopOrderCloseReq{
			Market:              opts.Symbol,
			OrderID:             newOrderID,
			StopLossPrice:       protection.StopLossPrice,
			StopLossPriceType:   protection.StopLossPriceType,
			TakeProfitPrice:     protection.TakeProfitPrice,
			TakeProfitPriceType: protection.TakeProfitPriceType,
		}); err != nil {
			return fmt.Errorf("order edited to %d but failed to reattach SL/TP: %w", resp.OrderID, err)
		}
		if err := verifyEditedOrderProtection(ctx, f.Client, opts.Symbol, newOrderID, protection); err != nil {
			return fmt.Errorf("order edited to %d but failed to verify reattached SL/TP: %w", resp.OrderID, err)
		}
		if refreshed, err := f.Client.Order.OrderDetail(ctx, futures.OrderDetailReq{
			Market: opts.Symbol, OrderID: newOrderID,
		}); err == nil {
			resp.OrderItem = *refreshed
		} else {
			resp.StopLossPrice = protection.StopLossPrice
			resp.TakeProfitPrice = protection.TakeProfitPrice
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
			{Key: "SL", Value: emptyDash(resp.StopLossPrice)},
			{Key: "TP", Value: emptyDash(resp.TakeProfitPrice)},
		})
	})
}

type orderProtection struct {
	StopLossPrice           string
	StopLossPriceType       futures.StopTriggerType
	TakeProfitPrice         string
	TakeProfitPriceType     futures.StopTriggerType
	ConflictsWithOtherOrder bool
}

func (p orderProtection) HasAny() bool {
	return priceSet(p.StopLossPrice) || priceSet(p.TakeProfitPrice)
}

func readOrderProtection(ctx context.Context, c *futures.Client, market, orderID string) (orderProtection, error) {
	order, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: market, OrderID: orderID})
	if err != nil {
		return orderProtection{}, err
	}
	p := orderProtection{}
	if priceSet(order.StopLossPrice) {
		p.StopLossPrice = order.StopLossPrice
		p.StopLossPriceType = futures.StopTriggerTypeLast
	}
	if priceSet(order.TakeProfitPrice) {
		p.TakeProfitPrice = order.TakeProfitPrice
		p.TakeProfitPriceType = futures.StopTriggerTypeLast
	}
	stops, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: market, Page: 1, PageSize: 100})
	if err != nil {
		return p, err
	}
	for _, s := range stops.Records {
		if s.PositionID == order.PositionID && strconv.FormatInt(s.OrderID, 10) != orderID {
			p.ConflictsWithOtherOrder = true
		}
		if strconv.FormatInt(s.OrderID, 10) != orderID {
			continue
		}
		switch s.ContractOrderType {
		case futures.StopOrderTypeOrderStopLoss:
			p.StopLossPrice = s.TriggerPrice
			p.StopLossPriceType = s.TriggerType
		case futures.StopOrderTypeOrderTakeProfit:
			p.TakeProfitPrice = s.TriggerPrice
			p.TakeProfitPriceType = s.TriggerType
		}
	}
	return p, nil
}

func verifyEditedOrderProtection(ctx context.Context, c *futures.Client, market, orderID string, p orderProtection) error {
	order, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: market, OrderID: orderID})
	if err != nil {
		return err
	}
	if priceSet(p.StopLossPrice) && order.StopLossPrice != p.StopLossPrice {
		return fmt.Errorf("SL is %q, want %q", order.StopLossPrice, p.StopLossPrice)
	}
	if priceSet(p.TakeProfitPrice) && order.TakeProfitPrice != p.TakeProfitPrice {
		return fmt.Errorf("TP is %q, want %q", order.TakeProfitPrice, p.TakeProfitPrice)
	}
	return nil
}

func priceSet(value string) bool {
	return value != "" && value != "-"
}
