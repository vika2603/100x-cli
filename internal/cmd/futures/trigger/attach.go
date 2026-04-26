package trigger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// NewCmdAttach is the `attach` group: `attach order` and `attach position`.
func NewCmdAttach(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach",
		Short: "Attach SL/TP to an existing order or position",
		Example: "# Attach a stop-loss leg at 68000 to a BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type SL --trigger-price 68000\n\n" +
			"# Attach a take-profit leg at 82000 to a BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type TP --trigger-price 82000",
	}
	c.AddCommand(NewCmdAttachOrder(f), NewCmdAttachPosition(f))
	return c
}

// AttachOrderOptions captures the flag-bound state of `trigger attach order`.
type AttachOrderOptions struct {
	Symbol       string
	OrderID      string
	Leg          string
	TriggerPrice string
	TriggerBy    string
	ClearOther   bool

	Factory *factory.Factory
}

// NewCmdAttachOrder builds the `trigger attach order` cobra command.
func NewCmdAttachOrder(f *factory.Factory) *cobra.Command {
	opts := &AttachOrderOptions{Factory: f}
	c := &cobra.Command{
		Use:   "order <symbol> <order-id>",
		Short: "Attach an SL or TP leg to a pending order",
		Long: "Attach an SL or TP leg to a pending order.\n\n" +
			"The gateway applies order-level SL/TP at position scope. The CLI preserves the opposite leg only when the backend can do so safely.",
		Example: "# Attach a stop-loss leg at 68000 to one pending BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type SL --trigger-price 68000\n\n" +
			"# Replace the opposite leg while attaching take-profit at 82000\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type TP --trigger-price 82000 --clear-other",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runAttachOrder(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leg, "type", "", "leg to set: SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "new trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "trigger feed: LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite SL/TP leg")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"SL", "TP"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("trigger-by", cobra.FixedCompletions([]string{"LAST", "INDEX", "MARK"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runAttachOrder(ctx context.Context, opts *AttachOrderOptions) error {
	leg, err := shared.ParseLeg(opts.Leg)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.TriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	order, stops, err := readOrderAttachState(ctx, f.Client, opts.Symbol, opts.OrderID)
	if err != nil {
		return err
	}
	if err := rejectCrossOrderTriggerConflict(order, stops); err != nil {
		return err
	}
	if !opts.ClearOther {
		existing := findOrderTriggerIn(stops, order.OrderID, leg)
		if existing != nil {
			if err := updateAttachedTrigger(ctx, f, opts.Symbol, existing.ContractOrderID, opts.TriggerPrice, priceType); err != nil {
				return err
			}
			return verifyOrderLeg(ctx, f.Client, opts.Symbol, opts.OrderID, leg, opts.TriggerPrice)
		}
	}
	if !opts.ClearOther && leg == shared.LegTP {
		if sl := findOrderTriggerIn(stops, order.OrderID, shared.LegSL); sl != nil {
			if err := attachTPPreservingOrderSL(ctx, f.Client, opts.Symbol, opts.OrderID, sl, opts.TriggerPrice, priceType); err != nil {
				return err
			}
			return f.IO.Render(futures.StopOrderCloseResp{}, func() error {
				return f.IO.Resultln("Attached", opts.Leg, "trigger to order", opts.OrderID)
			})
		}
	}
	body, err := shared.BuildAttachOrderReq(ctx, f.Client, shared.AttachOrderInput{
		Symbol: opts.Symbol, OrderID: opts.OrderID,
		Leg: leg, Price: opts.TriggerPrice, PriceType: priceType,
		ClearOther: opts.ClearOther,
	})
	if err != nil {
		return err
	}
	resp, err := f.Client.Order.StopOrderClose(ctx, body)
	if err != nil {
		return err
	}
	if err := verifyOrderLeg(ctx, f.Client, opts.Symbol, opts.OrderID, leg, opts.TriggerPrice); err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Attached", opts.Leg, "trigger to order", opts.OrderID)
	})
}

// AttachPositionOptions captures the flag-bound state of `trigger attach position`.
type AttachPositionOptions struct {
	Symbol       string
	PositionID   string
	Leg          string
	TriggerPrice string
	TriggerBy    string
	ClearOther   bool

	Factory *factory.Factory
}

// NewCmdAttachPosition builds the `trigger attach position` cobra command.
func NewCmdAttachPosition(f *factory.Factory) *cobra.Command {
	opts := &AttachPositionOptions{Factory: f}
	c := &cobra.Command{
		Use:   "position <symbol> <position-id>",
		Short: "Attach an SL or TP leg to an open position",
		Example: "# Attach a stop-loss leg at 68000 to one BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type SL --trigger-price 68000\n\n" +
			"# Attach a take-profit leg at 82000 to one BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type TP --trigger-price 82000",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.PositionID = args[1]
			return runAttachPosition(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leg, "type", "", "leg to set: SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "new trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "trigger feed: LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite SL/TP leg")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"SL", "TP"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("trigger-by", cobra.FixedCompletions([]string{"LAST", "INDEX", "MARK"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runAttachPosition(ctx context.Context, opts *AttachPositionOptions) error {
	leg, err := shared.ParseLeg(opts.Leg)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.TriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	if !opts.ClearOther {
		existing, err := findPositionTrigger(ctx, f.Client, opts.Symbol, opts.PositionID, leg)
		if err != nil {
			return err
		}
		if existing != nil {
			return updateAttachedTrigger(ctx, f, opts.Symbol, existing.ContractOrderID, opts.TriggerPrice, priceType)
		}
	}
	body, err := shared.BuildAttachPositionReq(ctx, f.Client, shared.AttachPositionInput{
		Symbol: opts.Symbol, PositionID: opts.PositionID,
		Leg: leg, Price: opts.TriggerPrice, PriceType: priceType,
		ClearOther: opts.ClearOther,
	})
	if err != nil {
		return err
	}
	resp, err := f.Client.Position.StopClosePosition(ctx, body)
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Attached", opts.Leg, "trigger to position on", opts.Symbol)
	})
}

func readOrderAttachState(ctx context.Context, c *futures.Client, symbol, orderID string) (*futures.OrderItem, []futures.StopOrderItem, error) {
	order, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: symbol, OrderID: orderID})
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: symbol, Page: 1, PageSize: 100})
	if err != nil {
		return nil, nil, err
	}
	return order, resp.Records, nil
}

func findOrderTriggerIn(stops []futures.StopOrderItem, orderID int64, leg shared.Leg) *futures.StopOrderItem {
	want := futures.StopOrderTypeOrderTakeProfit
	if leg == shared.LegSL {
		want = futures.StopOrderTypeOrderStopLoss
	}
	for i := range stops {
		if stops[i].ContractOrderType == want && stops[i].OrderID == orderID {
			return &stops[i]
		}
	}
	return nil
}

func rejectCrossOrderTriggerConflict(order *futures.OrderItem, stops []futures.StopOrderItem) error {
	for _, s := range stops {
		if s.PositionID == order.PositionID && s.OrderID != order.OrderID {
			return fmt.Errorf(
				"cannot attach trigger to order %d: existing trigger %s belongs to order %d on the same position; gateway applies order SL/TP by position, so edit/cancel that trigger first",
				order.OrderID, s.ContractOrderID, s.OrderID,
			)
		}
	}
	return nil
}

func attachTPPreservingOrderSL(ctx context.Context, c *futures.Client, symbol, orderID string, sl *futures.StopOrderItem, tpPrice string, tpType futures.StopTriggerType) error {
	if _, err := c.Order.StopOrderClose(ctx, futures.StopOrderCloseReq{
		Market:              symbol,
		OrderID:             orderID,
		TakeProfitPrice:     tpPrice,
		TakeProfitPriceType: tpType,
	}); err != nil {
		return err
	}
	if _, err := c.Order.StopOrderClose(ctx, futures.StopOrderCloseReq{
		Market:              symbol,
		OrderID:             orderID,
		StopLossPrice:       sl.TriggerPrice,
		StopLossPriceType:   sl.TriggerType,
		TakeProfitPrice:     tpPrice,
		TakeProfitPriceType: tpType,
	}); err != nil {
		return err
	}
	if err := verifyOrderLeg(ctx, c, symbol, orderID, shared.LegSL, sl.TriggerPrice); err != nil {
		return err
	}
	return verifyOrderLeg(ctx, c, symbol, orderID, shared.LegTP, tpPrice)
}

func verifyOrderLeg(ctx context.Context, c *futures.Client, symbol, orderID string, leg shared.Leg, want string) error {
	order, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: symbol, OrderID: orderID})
	if err != nil {
		return err
	}
	got := order.TakeProfitPrice
	name := "TP"
	if leg == shared.LegSL {
		got = order.StopLossPrice
		name = "SL"
	}
	if got != want {
		return fmt.Errorf("gateway accepted attach but %s on order %s is %q, want %q", name, orderID, got, want)
	}
	return nil
}

func findPositionTrigger(ctx context.Context, c *futures.Client, symbol, positionID string, leg shared.Leg) (*futures.StopOrderItem, error) {
	id, err := strconv.ParseInt(positionID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid position id %q", positionID)
	}
	want := futures.StopOrderTypePositionTakeProfit
	if leg == shared.LegSL {
		want = futures.StopOrderTypePositionStopLoss
	}
	return findAttachedTrigger(ctx, c, symbol, want, func(s futures.StopOrderItem) bool {
		return s.PositionID == id
	})
}

func findAttachedTrigger(ctx context.Context, c *futures.Client, symbol string, want futures.StopOrderType, match func(futures.StopOrderItem) bool) (*futures.StopOrderItem, error) {
	resp, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: symbol, Page: 1, PageSize: 100})
	if err != nil {
		return nil, err
	}
	for i := range resp.Records {
		s := resp.Records[i]
		if s.ContractOrderType == want && match(s) {
			return &s, nil
		}
	}
	return nil, nil
}

func updateAttachedTrigger(ctx context.Context, f *factory.Factory, symbol, triggerID, price string, priceType futures.StopTriggerType) error {
	resp, err := f.Client.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
		Market: symbol, StopOrderID: triggerID, StopPrice: price, StopPriceType: priceType,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Trigger ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Symbol", Value: symbol},
			{Key: "Trigger Price", Value: price},
			{Key: "Trigger By", Value: format.Enum(priceType.String())},
		})
	})
}
