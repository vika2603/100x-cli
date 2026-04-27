package position

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// PreferenceOptions captures the flag-bound state of `futures preference`.
type PreferenceOptions struct {
	Symbol   string
	Leverage string
	Mode     string

	Factory *factory.Factory
}

// NewCmdPreference builds the `futures preference` cobra command.
func NewCmdPreference(f *factory.Factory) *cobra.Command {
	opts := &PreferenceOptions{Factory: f}
	c := &cobra.Command{
		Use:     "preference <symbol>",
		Aliases: []string{"pref"},
		Short:   "Read or update per-market preferences",
		Long: "Read or update per-market preferences.\n\n" +
			"When updating only one field, the CLI preserves the other field automatically because the gateway expects both values together.",
		Example: "# Show the current leverage and margin mode for BTCUSDT\n" +
			"  100x futures preference BTCUSDT\n\n" +
			"# Set BTCUSDT leverage to 25 and preserve the current mode\n" +
			"  100x futures preference BTCUSDT --leverage 25\n\n" +
			"# Set BTCUSDT leverage to 25 and mode to CROSS\n" +
			"  100x futures preference BTCUSDT --leverage 25 --mode CROSS",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPreference(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leverage, "leverage", "", "Target leverage; keep current when omitted")
	c.Flags().StringVar(&opts.Mode, "mode", "", "Margin mode: ISOLATED | CROSS; keep current when omitted")
	_ = c.RegisterFlagCompletionFunc("mode", complete.MarginModes)
	return c
}

func runPreference(ctx context.Context, opts *PreferenceOptions) error {
	f := opts.Factory
	if err := clierr.PositiveNumber("--leverage", opts.Leverage); err != nil {
		return err
	}
	if opts.Leverage == "" && opts.Mode == "" {
		resp, err := f.Client.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: opts.Symbol})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Leverage", Value: resp.Leverage},
				{Key: "Mode", Value: format.PositionType(f.IO, resp.PositionType)},
			})
		})
	}
	req, err := buildAdjustMarketPreferenceReq(ctx, f.Client, mergedPreferenceInput{
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
			{Key: "Leverage", Value: updated.Leverage},
			{Key: "Mode", Value: format.PositionType(f.IO, updated.PositionType)},
		})
	})
}

func parsePositionType(s string) (futures.PositionType, error) {
	switch strings.ToUpper(s) {
	case "CROSS":
		return futures.PositionTypeCross, nil
	case "ISOLATED":
		return futures.PositionTypeIsolated, nil
	}
	return 0, clierr.Usagef("unknown mode %q (want ISOLATED|CROSS)", s)
}

type mergedPreferenceInput struct {
	Symbol       string
	Leverage     string
	PositionType string
}

// buildAdjustMarketPreferenceReq performs the read-modify-send compensation:
// the gateway's preference update takes leverage and position_type together,
// so a partial CLI update reads current values first and merges.
func buildAdjustMarketPreferenceReq(ctx context.Context, c *futures.Client, in mergedPreferenceInput) (futures.AdjustMarketPreferenceReq, error) {
	out := futures.AdjustMarketPreferenceReq{Market: in.Symbol}
	if in.Leverage == "" || in.PositionType == "" {
		cur, err := c.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: in.Symbol})
		if err != nil {
			return out, err
		}
		if in.Leverage == "" {
			out.Leverage = cur.Leverage
		}
		if in.PositionType == "" {
			out.PositionType = cur.PositionType
		}
	}
	if in.Leverage != "" {
		out.Leverage = in.Leverage
	}
	if in.PositionType != "" {
		pt, err := parsePositionType(in.PositionType)
		if err != nil {
			return out, err
		}
		out.PositionType = pt
	}
	return out, nil
}
