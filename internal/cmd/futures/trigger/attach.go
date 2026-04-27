package trigger

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/cmd/futures/protection"
)

// NewCmdAttach is the `attach` group: `attach order` and `attach position`.
func NewCmdAttach(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach",
		Short: "Attach SL/TP to an existing order or position",
		Example: "# Attach a stop-loss at 68000 to a BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type SL --trigger-price 68000\n\n" +
			"# Attach a take-profit at 82000 to a BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type TP --trigger-price 82000",
	}
	c.AddCommand(NewCmdAttachOrder(f), NewCmdAttachPosition(f))
	return c
}

// AttachOrderOptions captures the flag-bound state of `trigger attach order`.
type AttachOrderOptions struct {
	Symbol       string
	OrderID      string
	Kind         string
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
		Short: "Attach an SL or TP to a pending order",
		Long: "Attach an SL or TP to a pending order.\n\n" +
			"The gateway applies order-level SL/TP at position scope. The CLI preserves the unchanged side only when the backend can do so safely.",
		Example: "# Attach a stop-loss at 68000 to one pending BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type SL --trigger-price 68000\n\n" +
			"# Replace the unchanged side while attaching take-profit at 82000\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --type TP --trigger-price 82000 --clear-other",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenOrderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runAttachOrder(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Kind, "type", "", "Which side to set: SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "New trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Trigger feed: LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "Clear the unchanged SL/TP side instead of preserving it")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("type", complete.TriggerLegs)
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	return c
}

func runAttachOrder(ctx context.Context, opts *AttachOrderOptions) error {
	if err := clierr.PositiveID("order-id", opts.OrderID); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--trigger-price", opts.TriggerPrice); err != nil {
		return err
	}
	kind, err := protection.ParseKind(opts.Kind)
	if err != nil {
		return err
	}
	var priceType futures.StopTriggerType
	switch strings.ToUpper(opts.TriggerBy) {
	case "", "LAST":
		priceType = futures.StopTriggerTypeLast
	case "INDEX":
		priceType = futures.StopTriggerTypeIndex
	case "MARK":
		priceType = futures.StopTriggerTypeMark
	default:
		return clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", opts.TriggerBy)
	}
	f := opts.Factory
	target := protection.Order{Symbol: opts.Symbol, OrderID: opts.OrderID}
	current, err := target.Inspect(ctx, f.Client)
	if err != nil {
		return err
	}
	if current.CrossOrderConflict {
		return fmt.Errorf("cannot attach trigger to order %s: another pending order on the same position has active triggers; gateway applies order SL/TP by position, so edit/cancel that trigger first", opts.OrderID)
	}
	want := planAttach(current, kind, opts.TriggerPrice, priceType, opts.ClearOther)
	if err := target.Apply(ctx, f.Client, current, want); err != nil {
		return err
	}
	if err := target.Verify(ctx, f.Client, want); err != nil {
		return err
	}
	return f.IO.Resultln("Attached", kind.String(), "to order", opts.OrderID)
}

// AttachPositionOptions captures the flag-bound state of `trigger attach position`.
type AttachPositionOptions struct {
	Symbol       string
	PositionID   string
	Kind         string
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
		Short: "Attach an SL or TP to an open position",
		Example: "# Attach a stop-loss at 68000 to one BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type SL --trigger-price 68000\n\n" +
			"# Attach a take-profit at 82000 to one BTCUSDT position\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --type TP --trigger-price 82000",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenPositionArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.PositionID = args[1]
			return runAttachPosition(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Kind, "type", "", "Which side to set: SL | TP")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "New trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Trigger feed: LAST | INDEX | MARK")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "Clear the unchanged SL/TP side instead of preserving it")
	_ = c.MarkFlagRequired("type")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("type", complete.TriggerLegs)
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	return c
}

func runAttachPosition(ctx context.Context, opts *AttachPositionOptions) error {
	if err := clierr.PositiveID("position-id", opts.PositionID); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--trigger-price", opts.TriggerPrice); err != nil {
		return err
	}
	kind, err := protection.ParseKind(opts.Kind)
	if err != nil {
		return err
	}
	var priceType futures.StopTriggerType
	switch strings.ToUpper(opts.TriggerBy) {
	case "", "LAST":
		priceType = futures.StopTriggerTypeLast
	case "INDEX":
		priceType = futures.StopTriggerTypeIndex
	case "MARK":
		priceType = futures.StopTriggerTypeMark
	default:
		return clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", opts.TriggerBy)
	}
	f := opts.Factory
	target := protection.Position{Symbol: opts.Symbol, PositionID: opts.PositionID}
	current, err := target.Inspect(ctx, f.Client)
	if err != nil {
		return err
	}
	want := planAttach(current, kind, opts.TriggerPrice, priceType, opts.ClearOther)
	if err := target.Apply(ctx, f.Client, current, want); err != nil {
		return err
	}
	return f.IO.Resultln("Attached", kind.String(), "to position on", opts.Symbol)
}

// planAttach derives the desired State for an attach: starting from the
// current state it overwrites the requested side with the new price and
// either preserves or clears the unchanged side per clearOther.
func planAttach(current protection.State, k protection.Kind, price string, priceType futures.StopTriggerType, clearOther bool) protection.State {
	want := current
	switch k {
	case protection.KindStopLoss:
		want.SL = protection.Stop{Price: price, PriceType: priceType, TriggerID: current.SL.TriggerID}
		if clearOther {
			want.TP = protection.Stop{}
		}
	case protection.KindTakeProfit:
		want.TP = protection.Stop{Price: price, PriceType: priceType, TriggerID: current.TP.TriggerID}
		if clearOther {
			want.SL = protection.Stop{}
		}
	}
	return want
}
