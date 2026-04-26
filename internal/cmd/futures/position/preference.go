package position

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position/shared"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// PreferenceOptions captures the flag-bound state of `position preference`.
type PreferenceOptions struct {
	Symbol   string
	Leverage string
	Mode     string

	Factory *factory.Factory
}

// NewCmdPreference builds the `position preference` cobra command.
func NewCmdPreference(f *factory.Factory) *cobra.Command {
	opts := &PreferenceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "preference <symbol>",
		Short: "Read or update per-market preferences",
		Long: "Read or update per-market preferences.\n\n" +
			"When updating only one field, the CLI preserves the other field automatically because the gateway expects both values together.",
		Example: "# Show the current leverage and margin mode for BTCUSDT\n" +
			"  100x futures position preference BTCUSDT\n\n" +
			"# Set BTCUSDT leverage to 25 and preserve the current mode\n" +
			"  100x futures position preference BTCUSDT --leverage 25\n\n" +
			"# Set BTCUSDT leverage to 25 and mode to CROSS\n" +
			"  100x futures position preference BTCUSDT --leverage 25 --mode CROSS",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPreference(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leverage, "leverage", "", "target leverage; keep current when omitted")
	c.Flags().StringVar(&opts.Mode, "mode", "", "margin mode: ISOLATED | CROSS; keep current when omitted")
	_ = c.RegisterFlagCompletionFunc("mode", cobra.FixedCompletions([]string{"ISOLATED", "CROSS"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runPreference(ctx context.Context, opts *PreferenceOptions) error {
	f := opts.Factory
	if opts.Leverage == "" && opts.Mode == "" {
		resp, err := f.Client.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: opts.Symbol})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Symbol", Value: opts.Symbol},
				{Key: "Leverage", Value: resp.Leverage},
				{Key: "Mode", Value: format.PositionType(f.IO, resp.PositionType)},
			})
		})
	}
	req, err := shared.BuildAdjustMarketPreferenceReq(ctx, f.Client, shared.MergedPreferenceInput{
		Symbol: opts.Symbol, Leverage: opts.Leverage, PositionType: opts.Mode,
	})
	if err != nil {
		return err
	}
	if _, err := f.Client.Setting.AdjustMarketPreference(ctx, req); err != nil {
		return err
	}
	updated, err := f.Client.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(updated, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Symbol", Value: opts.Symbol},
			{Key: "Leverage", Value: updated.Leverage},
			{Key: "Mode", Value: format.PositionType(f.IO, updated.PositionType)},
		})
	})
}
