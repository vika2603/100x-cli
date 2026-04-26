package position

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position/shared"
)

// LeverageOptions captures the flag-bound state of `position leverage`.
type LeverageOptions struct {
	Market   string
	Leverage string

	Factory *factory.Factory
}

// NewCmdLeverage builds `position leverage <value>`, a thin convenience over
// `preference --leverage`.
func NewCmdLeverage(f *factory.Factory) *cobra.Command {
	opts := &LeverageOptions{Factory: f}
	c := &cobra.Command{
		Use:   "leverage <value>",
		Short: "Set leverage for one market (preserves position type)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Leverage = args[0]
			return runLeverage(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

func runLeverage(ctx context.Context, opts *LeverageOptions) error {
	f := opts.Factory
	req, err := shared.BuildAdjustMarketPreferenceReq(ctx, f.Client, shared.MergedPreferenceInput{
		Market: opts.Market, Leverage: opts.Leverage,
	})
	if err != nil {
		return err
	}
	resp, err := f.Client.Setting.AdjustMarketPreference(ctx, req)
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("leverage set to", opts.Leverage, "for", opts.Market)
		return nil
	})
}
