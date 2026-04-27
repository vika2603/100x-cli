package position

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/output"
)

// MarginOptions captures the flag-bound state of `position margin`.
type MarginOptions struct {
	Symbol     string
	PositionID string
	Add        string
	Reduce     string

	Factory *factory.Factory
}

// NewCmdMargin builds the `position margin` cobra command.
func NewCmdMargin(f *factory.Factory) *cobra.Command {
	opts := &MarginOptions{Factory: f}
	c := &cobra.Command{
		Use:   "margin <symbol>",
		Short: "Read or adjust isolated-position margin",
		Long: "Read or adjust isolated-position margin.\n\n" +
			"Without --add or --reduce, the command reads current adjustable margin. With either flag, it performs a write.",
		Example: "# Read adjustable isolated margin for one BTCUSDT position\n" +
			"  100x futures position margin BTCUSDT --position-id <position-id>\n\n" +
			"# Add 10 units of isolated margin to the BTCUSDT position\n" +
			"  100x futures position margin BTCUSDT --add 10\n\n" +
			"# Remove 5 units of isolated margin from the BTCUSDT position\n" +
			"  100x futures position margin BTCUSDT --reduce 5",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runMargin(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "Position ID for read mode; required when the symbol matches multiple positions")
	c.Flags().StringVar(&opts.Add, "add", "", "Amount of isolated margin to add")
	c.Flags().StringVar(&opts.Reduce, "reduce", "", "Amount of isolated margin to remove")
	_ = c.RegisterFlagCompletionFunc("position-id", complete.OpenPositionIDs)
	return c
}

func runMargin(ctx context.Context, opts *MarginOptions) error {
	f := opts.Factory
	if err := clierr.PositiveID("position-id", opts.PositionID); opts.PositionID != "" && err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--add", opts.Add); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--reduce", opts.Reduce); err != nil {
		return err
	}
	if opts.Add != "" && opts.Reduce != "" {
		return clierr.Usagef("--add and --reduce are mutually exclusive")
	}
	if opts.Add == "" && opts.Reduce == "" {
		return renderMargin(ctx, f, opts.Symbol, opts.PositionID)
	}
	if opts.PositionID != "" {
		return clierr.Usagef("--position-id is only valid in read mode")
	}

	pref, err := f.Client.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	if pref.PositionType == futures.PositionTypeCross {
		return clierr.Usagef("position margin adjust requires ISOLATED mode; %s is in CROSS", opts.Symbol)
	}

	action, amount := futures.MarginActionAdd, opts.Add
	if opts.Reduce != "" {
		action, amount = futures.MarginActionRemove, opts.Reduce
	}
	if _, err := f.Client.Position.AdjustPositionMargin(ctx, futures.AdjustPositionMarginReq{
		Market: opts.Symbol, Type: action, Quantity: amount,
	}); err != nil {
		return err
	}
	return renderMargin(ctx, f, opts.Symbol, "")
}

func renderMargin(ctx context.Context, f *factory.Factory, market, positionID string) error {
	resolved, err := resolvePositionID(ctx, f.Client, market, positionID)
	if err != nil {
		return err
	}
	id, err := strconv.Atoi(resolved)
	if err != nil {
		return err
	}
	resp, err := f.Client.Position.PositionAdjustableMargin(ctx, futures.PositionAdjustableMarginReq{Market: market, PositionID: id})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		// PositionAdjustableMarginResp has no market field; user already
		// passed the symbol on the command line, so omit it here rather
		// than echoing whatever client-side string we happened to send.
		return f.IO.Object([]output.KV{
			{Key: "Position ID", Value: resolved},
			{Key: "Leverage", Value: resp.Leverage + "x"},
			{Key: "Amount", Value: resp.Amount},
			{Key: "Margin Amount", Value: resp.MarginAmount},
			{Key: "Available", Value: resp.Available},
			{Key: "Max Removable Margin", Value: resp.MaxRemovableMargin},
		})
	})
}
