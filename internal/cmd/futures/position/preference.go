package position

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position/shared"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPreference(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Leverage, "leverage", "", "leverage (preserve when omitted)")
	c.Flags().StringVar(&opts.Mode, "mode", "", "ISOLATED | CROSS (preserve when omitted)")
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
				{Key: "Mode", Value: style.PositionType(f.IO, resp.PositionType)},
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
			{Key: "Mode", Value: style.PositionType(f.IO, updated.PositionType)},
		})
	})
}
