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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Price, "price", "", "new price")
	c.Flags().StringVar(&opts.Size, "size", "", "new size")
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
			{Key: "Side", Value: style.Side(f.IO, resp.Side)},
			{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
			{Key: "Price", Value: resp.Price},
			{Key: "Size", Value: resp.Volume},
			{Key: "Filled", Value: resp.Filled},
			{Key: "SL", Value: emptyDash(resp.StopLossPrice)},
			{Key: "TP", Value: emptyDash(resp.TakeProfitPrice)},
		})
	})
}

type orderProtection struct {
	StopLossPrice       string
	StopLossPriceType   futures.StopTriggerType
	TakeProfitPrice     string
	TakeProfitPriceType futures.StopTriggerType
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
		return p, nil
	}
	for _, s := range stops.Records {
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

func priceSet(value string) bool {
	return value != "" && value != "-"
}
