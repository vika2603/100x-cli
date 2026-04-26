package position

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/position/shared"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// MarginOptions captures the flag-bound state of `position margin`.
type MarginOptions struct {
	Market         string
	Action         string
	Quantity       string
	ShowAdjustable bool

	Factory *factory.Factory
}

// NewCmdMargin builds the `position margin` cobra command.
func NewCmdMargin(f *factory.Factory) *cobra.Command {
	opts := &MarginOptions{Factory: f}
	c := &cobra.Command{
		Use:   "margin",
		Short: "Adjust isolated-position margin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMargin(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Action, "action", "", "add | remove")
	c.Flags().StringVar(&opts.Quantity, "qty", "", "amount to move")
	c.Flags().BoolVar(&opts.ShowAdjustable, "show-adjustable", false, "query margin currently movable")
	_ = c.MarkFlagRequired("market")
	_ = c.RegisterFlagCompletionFunc("action", cobra.FixedCompletions([]string{"add", "remove"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runMargin(ctx context.Context, opts *MarginOptions) error {
	f := opts.Factory
	if opts.ShowAdjustable {
		resp, err := f.Client.Position.PositionAdjustableMargin(ctx, futures.PositionAdjustableMarginReq{Market: opts.Market})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Market", Value: opts.Market},
				{Key: "Leverage", Value: resp.Leverage},
				{Key: "Margin", Value: resp.MarginAmount},
				{Key: "Amount", Value: resp.Amount},
				{Key: "Available", Value: resp.Available},
				{Key: "Max Removable", Value: resp.MaxRemovableMargin},
			})
		})
	}
	a, err := shared.ParseMarginAction(opts.Action)
	if err != nil {
		return err
	}
	resp, err := f.Client.Position.AdjustPositionMargin(ctx, futures.AdjustPositionMarginReq{
		Market: opts.Market, Type: a, Quantity: opts.Quantity,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Position ID", Value: strconv.Itoa(resp.PositionID)},
			{Key: "Market", Value: resp.Market},
			{Key: "Side", Value: style.Side(f.IO, resp.Side)},
			{Key: "Margin", Value: resp.MarginAmount},
			{Key: "Qty", Value: resp.Volume},
			{Key: "Action", Value: strings.ToUpper(opts.Action)},
		})
	})
}
