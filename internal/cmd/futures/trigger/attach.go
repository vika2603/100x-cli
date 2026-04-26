package trigger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
	"github.com/vika2603/100x-cli/internal/output"
)

// NewCmdAttach is the `attach` group: `attach order` and `attach position`.
func NewCmdAttach(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach",
		Short: "Attach SL/TP to an existing order or position",
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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runAttachOrder(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leg, "type", "", "SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite leg")
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
	if !opts.ClearOther {
		existing, err := findOrderTrigger(ctx, f.Client, opts.Symbol, opts.OrderID, leg)
		if err != nil {
			return err
		}
		if existing != nil {
			return updateAttachedTrigger(ctx, f, opts.Symbol, existing.ContractOrderID, opts.TriggerPrice, priceType)
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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.PositionID = args[1]
			return runAttachPosition(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leg, "type", "", "SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite leg")
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

func findOrderTrigger(ctx context.Context, c *futures.Client, symbol, orderID string, leg shared.Leg) (*futures.StopOrderItem, error) {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order id %q", orderID)
	}
	want := futures.StopOrderTypeOrderTakeProfit
	if leg == shared.LegSL {
		want = futures.StopOrderTypeOrderStopLoss
	}
	return findAttachedTrigger(ctx, c, symbol, want, func(s futures.StopOrderItem) bool {
		return s.OrderID == id
	})
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
			{Key: "Trigger By", Value: priceType.String()},
		})
	})
}
