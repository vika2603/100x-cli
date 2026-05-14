package trigger

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/cmd/futures/protection"
	"github.com/vika2603/100x-cli/internal/wire"
)

// attachStop is the per-side projection of protection.Stop in the attach
// command's JSON payload.
type attachStop struct {
	Price     string `json:"price"`
	TriggerBy string `json:"trigger_by"`
}

// attachOrderResult is the JSON payload of `trigger attach order`.
type attachOrderResult struct {
	Symbol  string      `json:"symbol"`
	OrderID string      `json:"order_id"`
	SL      *attachStop `json:"sl,omitempty"`
	TP      *attachStop `json:"tp,omitempty"`
}

// attachPositionResult is the JSON payload of `trigger attach position`.
type attachPositionResult struct {
	Symbol     string      `json:"symbol"`
	PositionID string      `json:"position_id"`
	SL         *attachStop `json:"sl,omitempty"`
	TP         *attachStop `json:"tp,omitempty"`
}

// stopOut projects a protection.Stop into the JSON shape, returning nil
// when the side carries no real price.
func stopOut(s protection.Stop) *attachStop {
	if !s.Set() {
		return nil
	}
	return &attachStop{Price: s.Price, TriggerBy: s.PriceType.String()}
}

// NewCmdAttach is the `attach` group: `attach order` and `attach position`.
func NewCmdAttach(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach",
		Short: "Attach SL/TP to an existing order or position",
		Example: "# Attach a stop-loss at 68000 to a BTCUSDT order\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --sl-price 68000\n\n" +
			"# Attach SL (mark feed) and TP (last feed) together in one call\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> \\\n" +
			"      --sl-price 68000 --sl-trigger-by MARK \\\n" +
			"      --tp-price 82000 --tp-trigger-by LAST",
	}
	c.AddCommand(NewCmdAttachOrder(f), NewCmdAttachPosition(f))
	return c
}

// AttachOrderOptions captures the flag-bound state of `trigger attach order`.
type AttachOrderOptions struct {
	Symbol      string
	OrderID     string
	SLPrice     string
	TPPrice     string
	TriggerBy   string
	SLTriggerBy string
	TPTriggerBy string
	ClearOther  bool

	Factory *factory.Factory
}

// NewCmdAttachOrder builds the `trigger attach order` cobra command.
func NewCmdAttachOrder(f *factory.Factory) *cobra.Command {
	opts := &AttachOrderOptions{Factory: f}
	c := &cobra.Command{
		Use:   "order <symbol> <order-id>",
		Short: "Attach SL and/or TP to a pending order",
		Long: "Attach SL and/or TP to a pending order.\n\n" +
			"Pass --sl-price, --tp-price, or both. With one side given, the other is preserved\n" +
			"automatically; pass --clear-other to drop it instead. With both given, the call sets\n" +
			"SL and TP in a single request and --clear-other is rejected as redundant.\n\n" +
			"--trigger-by sets the feed for both sides (default LAST). --sl-trigger-by and\n" +
			"--tp-trigger-by override that for one side, useful when SL and TP need different\n" +
			"feeds (e.g. SL on MARK, TP on LAST).\n\n" +
			"Order-level SL and TP are shared across all open orders on the same position.\n" +
			"The unchanged side is preserved automatically when there is no conflict; otherwise\n" +
			"the command errors so you can edit or cancel the conflicting trigger first.",
		Example: "# Attach a stop-loss at 68000 (LAST feed by default)\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --sl-price 68000\n\n" +
			"# Set SL and TP together in one request, both on the LAST feed\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> \\\n" +
			"      --sl-price 68000 --tp-price 82000\n\n" +
			"# Set SL on MARK and TP on LAST in one request\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> \\\n" +
			"      --sl-price 68000 --sl-trigger-by MARK \\\n" +
			"      --tp-price 82000 --tp-trigger-by LAST\n\n" +
			"# Replace the unchanged side while attaching take-profit at 82000\n" +
			"  100x futures trigger attach order BTCUSDT <order-id> --tp-price 82000 --clear-other",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenOrderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runAttachOrder(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.SLPrice, "sl-price", "", "Stop-loss trigger price")
	c.Flags().StringVar(&opts.TPPrice, "tp-price", "", "Take-profit trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Default trigger feed for both sides: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.SLTriggerBy, "sl-trigger-by", "", "Override SL trigger feed (defaults to --trigger-by)")
	c.Flags().StringVar(&opts.TPTriggerBy, "tp-trigger-by", "", "Override TP trigger feed (defaults to --trigger-by)")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "Clear the unspecified SL/TP side instead of preserving it")
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("sl-trigger-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("tp-trigger-by", complete.TriggerFeeds)
	return c
}

func runAttachOrder(ctx context.Context, opts *AttachOrderOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	if err := clierr.PositiveID("order-id", opts.OrderID); err != nil {
		return err
	}
	if err := validateSidePrices(opts.SLPrice, opts.TPPrice, opts.ClearOther); err != nil {
		return err
	}
	slType, tpType, err := parseSideTriggerBys(opts.TriggerBy, opts.SLTriggerBy, opts.TPTriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	target := protection.Order{Symbol: opts.Symbol, OrderID: opts.OrderID}
	current, err := target.Inspect(ctx, client)
	if err != nil {
		return err
	}
	if current.CrossOrderConflict {
		return fmt.Errorf("cannot attach trigger to order %s: another pending order on the same position has active triggers; gateway applies order SL/TP by position, so edit/cancel that trigger first", opts.OrderID)
	}
	want := planAttachSides(current, opts.SLPrice, opts.TPPrice, slType, tpType, opts.ClearOther)
	if err := target.Apply(ctx, client, current, want); err != nil {
		return err
	}
	if err := target.Verify(ctx, client, want); err != nil {
		return err
	}
	payload := attachOrderResult{
		Symbol:  opts.Symbol,
		OrderID: opts.OrderID,
		SL:      stopOut(want.SL),
		TP:      stopOut(want.TP),
	}
	return f.IO.Render(payload, func() error {
		return f.IO.Resultln("Attached", changedSideLabel(opts.SLPrice, opts.TPPrice), "to order", opts.OrderID)
	})
}

// AttachPositionOptions captures the flag-bound state of `trigger attach position`.
type AttachPositionOptions struct {
	Symbol      string
	PositionID  string
	SLPrice     string
	TPPrice     string
	TriggerBy   string
	SLTriggerBy string
	TPTriggerBy string
	ClearOther  bool

	Factory *factory.Factory
}

// NewCmdAttachPosition builds the `trigger attach position` cobra command.
func NewCmdAttachPosition(f *factory.Factory) *cobra.Command {
	opts := &AttachPositionOptions{Factory: f}
	c := &cobra.Command{
		Use:   "position <symbol> <position-id>",
		Short: "Attach SL and/or TP to an open position",
		Long: "Attach SL and/or TP to an open position.\n\n" +
			"Pass --sl-price, --tp-price, or both. With one side given, the other is preserved\n" +
			"automatically; pass --clear-other to drop it instead. With both given, the call sets\n" +
			"SL and TP in a single request and --clear-other is rejected as redundant.\n\n" +
			"--trigger-by sets the feed for both sides (default LAST). --sl-trigger-by and\n" +
			"--tp-trigger-by override that for one side, useful when SL and TP need different\n" +
			"feeds (e.g. SL on MARK, TP on LAST).",
		Example: "# Attach a stop-loss at 68000 (LAST feed)\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --sl-price 68000\n\n" +
			"# SL and TP together with different feeds\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> \\\n" +
			"      --sl-price 68000 --sl-trigger-by MARK \\\n" +
			"      --tp-price 82000 --tp-trigger-by LAST\n\n" +
			"# Replace the unchanged side while attaching take-profit at 82000\n" +
			"  100x futures trigger attach position BTCUSDT <position-id> --tp-price 82000 --clear-other",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.OpenPositionArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.PositionID = args[1]
			return runAttachPosition(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.SLPrice, "sl-price", "", "Stop-loss trigger price")
	c.Flags().StringVar(&opts.TPPrice, "tp-price", "", "Take-profit trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Default trigger feed for both sides: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.SLTriggerBy, "sl-trigger-by", "", "Override SL trigger feed (defaults to --trigger-by)")
	c.Flags().StringVar(&opts.TPTriggerBy, "tp-trigger-by", "", "Override TP trigger feed (defaults to --trigger-by)")
	c.Flags().BoolVar(&opts.ClearOther, "clear-other", false, "Clear the unspecified SL/TP side instead of preserving it")
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("sl-trigger-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("tp-trigger-by", complete.TriggerFeeds)
	return c
}

func runAttachPosition(ctx context.Context, opts *AttachPositionOptions) error {
	if err := clierr.PositiveID("position-id", opts.PositionID); err != nil {
		return err
	}
	if err := validateSidePrices(opts.SLPrice, opts.TPPrice, opts.ClearOther); err != nil {
		return err
	}
	slType, tpType, err := parseSideTriggerBys(opts.TriggerBy, opts.SLTriggerBy, opts.TPTriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	target := protection.Position{Symbol: opts.Symbol, PositionID: opts.PositionID}
	current, err := target.Inspect(ctx, client)
	if err != nil {
		return err
	}
	want := planAttachSides(current, opts.SLPrice, opts.TPPrice, slType, tpType, opts.ClearOther)
	if err := target.Apply(ctx, client, current, want); err != nil {
		return err
	}
	payload := attachPositionResult{
		Symbol:     opts.Symbol,
		PositionID: opts.PositionID,
		SL:         stopOut(want.SL),
		TP:         stopOut(want.TP),
	}
	return f.IO.Render(payload, func() error {
		return f.IO.Resultln("Attached", changedSideLabel(opts.SLPrice, opts.TPPrice), "to position on", opts.Symbol)
	})
}

// validateSidePrices enforces the attach flag invariants:
//   - at least one of --sl-price / --tp-price must be set
//   - given prices must be positive numbers
//   - --clear-other only makes sense when exactly one side is given
func validateSidePrices(slPrice, tpPrice string, clearOther bool) error {
	if slPrice == "" && tpPrice == "" {
		return clierr.Usagef("provide --sl-price, --tp-price, or both")
	}
	if slPrice != "" {
		if err := clierr.PositiveNumber("--sl-price", slPrice); err != nil {
			return err
		}
	}
	if tpPrice != "" {
		if err := clierr.PositiveNumber("--tp-price", tpPrice); err != nil {
			return err
		}
	}
	if clearOther && slPrice != "" && tpPrice != "" {
		return clierr.Usagef("--clear-other is meaningless when both --sl-price and --tp-price are given")
	}
	return nil
}

// planAttachSides derives the desired State from current plus whichever
// sides the caller wants to set. Unspecified sides are preserved (carrying
// any TriggerID through to Apply for routing) unless clearOther is set.
func planAttachSides(current protection.State, slPrice, tpPrice string, slType, tpType futures.StopTriggerType, clearOther bool) protection.State {
	want := current
	if slPrice != "" {
		want.SL = protection.Stop{Price: slPrice, PriceType: slType, TriggerID: current.SL.TriggerID}
	}
	if tpPrice != "" {
		want.TP = protection.Stop{Price: tpPrice, PriceType: tpType, TriggerID: current.TP.TriggerID}
	}
	if clearOther {
		if slPrice != "" && tpPrice == "" {
			want.TP = protection.Stop{}
		}
		if tpPrice != "" && slPrice == "" {
			want.SL = protection.Stop{}
		}
	}
	return want
}

// changedSideLabel produces a human-readable summary of which sides the
// command set, used in the success line.
func changedSideLabel(slPrice, tpPrice string) string {
	switch {
	case slPrice != "" && tpPrice != "":
		return "SL+TP"
	case slPrice != "":
		return "SL"
	default:
		return "TP"
	}
}

// parseSideTriggerBys resolves per-side trigger feeds from the hybrid
// flag set: a per-side override wins when given, otherwise both sides
// fall back to the common --trigger-by value.
func parseSideTriggerBys(common, slOverride, tpOverride string) (futures.StopTriggerType, futures.StopTriggerType, error) {
	commonType, err := futures.ParseStopTriggerType(common)
	if err != nil {
		return 0, 0, clierr.Usage(err)
	}
	slType := commonType
	if slOverride != "" {
		slType, err = futures.ParseStopTriggerType(slOverride)
		if err != nil {
			return 0, 0, clierr.Usage(err)
		}
	}
	tpType := commonType
	if tpOverride != "" {
		tpType, err = futures.ParseStopTriggerType(tpOverride)
		if err != nil {
			return 0, 0, clierr.Usage(err)
		}
	}
	return slType, tpType, nil
}
