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
	Market       string
	Leverage     string
	PositionType string

	Factory *factory.Factory
}

// NewCmdPreference builds the `position preference` cobra command.
func NewCmdPreference(f *factory.Factory) *cobra.Command {
	opts := &PreferenceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "preference",
		Short: "Read or update per-market preferences",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPreference(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Leverage, "leverage", "", "leverage (preserve when omitted)")
	c.Flags().StringVar(&opts.PositionType, "position-type", "", "cross | isolated (preserve when omitted)")
	_ = c.MarkFlagRequired("market")
	_ = c.RegisterFlagCompletionFunc("position-type", cobra.FixedCompletions([]string{"cross", "isolated"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runPreference(ctx context.Context, opts *PreferenceOptions) error {
	f := opts.Factory
	if opts.Leverage == "" && opts.PositionType == "" {
		resp, err := f.Client.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: opts.Market})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Market", Value: opts.Market},
				{Key: "Leverage", Value: resp.Leverage},
				{Key: "Position Type", Value: style.PositionType(f.IO, resp.PositionType)},
			})
		})
	}
	req, err := shared.BuildAdjustMarketPreferenceReq(ctx, f.Client, shared.MergedPreferenceInput{
		Market: opts.Market, Leverage: opts.Leverage, PositionType: opts.PositionType,
	})
	if err != nil {
		return err
	}
	resp, err := f.Client.Setting.AdjustMarketPreference(ctx, req)
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("preference updated for", opts.Market)
		return nil
	})
}
