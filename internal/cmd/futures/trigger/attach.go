package trigger

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
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
	Market     string
	OrderID    string
	Leg        string
	Price      string
	PriceType  string
	ClearOther bool

	Factory *factory.Factory
}

// NewCmdAttachOrder builds the `trigger attach order` cobra command.
func NewCmdAttachOrder(f *factory.Factory) *cobra.Command {
	opts := &AttachOrderOptions{Factory: f}
	c := &cobra.Command{
		Use:   "order <order-id>",
		Short: "Attach an SL or TP leg to a pending order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderID = args[0]
			return runAttachOrder(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Leg, "type", "", "SL | TP")
	c.Flags().StringVar(&opts.Price, "price", "", "trigger price")
	c.Flags().StringVar(&opts.PriceType, "price-type", "last", "last | index | mark")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite leg")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("price")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"SL", "TP"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("price-type", cobra.FixedCompletions([]string{"last", "index", "mark"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runAttachOrder(ctx context.Context, opts *AttachOrderOptions) error {
	leg, err := shared.ParseLeg(opts.Leg)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.PriceType)
	if err != nil {
		return err
	}
	f := opts.Factory
	body, err := shared.BuildAttachOrderReq(ctx, f.Client, shared.AttachOrderInput{
		Market: opts.Market, OrderID: opts.OrderID,
		Leg: leg, Price: opts.Price, PriceType: priceType,
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
		f.IO.Println("attached", opts.Leg, "to", opts.Market)
		return nil
	})
}

// AttachPositionOptions captures the flag-bound state of `trigger attach position`.
type AttachPositionOptions struct {
	Market     string
	PositionID string
	Leg        string
	Price      string
	PriceType  string
	ClearOther bool

	Factory *factory.Factory
}

// NewCmdAttachPosition builds the `trigger attach position` cobra command.
func NewCmdAttachPosition(f *factory.Factory) *cobra.Command {
	opts := &AttachPositionOptions{Factory: f}
	c := &cobra.Command{
		Use:   "position <position-id>",
		Short: "Attach an SL or TP leg to an open position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PositionID = args[0]
			return runAttachPosition(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Leg, "type", "", "SL | TP")
	c.Flags().StringVar(&opts.Price, "price", "", "trigger price")
	c.Flags().StringVar(&opts.PriceType, "price-type", "last", "last | index | mark")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "clear the opposite leg")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("price")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"SL", "TP"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("price-type", cobra.FixedCompletions([]string{"last", "index", "mark"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runAttachPosition(ctx context.Context, opts *AttachPositionOptions) error {
	leg, err := shared.ParseLeg(opts.Leg)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.PriceType)
	if err != nil {
		return err
	}
	f := opts.Factory
	body, err := shared.BuildAttachPositionReq(ctx, f.Client, shared.AttachPositionInput{
		Market: opts.Market, PositionID: opts.PositionID,
		Leg: leg, Price: opts.Price, PriceType: priceType,
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
		f.IO.Println("attached", opts.Leg, "to", opts.Market)
		return nil
	})
}
