package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runMargin(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "position id for read mode")
	c.Flags().StringVar(&opts.Add, "add", "", "margin amount to add")
	c.Flags().StringVar(&opts.Reduce, "reduce", "", "margin amount to reduce")
	return c
}

func runMargin(ctx context.Context, opts *MarginOptions) error {
	f := opts.Factory
	if opts.Add != "" && opts.Reduce != "" {
		return fmt.Errorf("--add and --reduce are mutually exclusive")
	}
	if opts.Add == "" && opts.Reduce == "" {
		return renderMargin(ctx, f, opts.Symbol, opts.PositionID)
	}
	if opts.PositionID != "" {
		return fmt.Errorf("--position-id is only valid in read mode")
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
		return f.IO.Object([]output.KV{
			{Key: "Symbol", Value: market},
			{Key: "Leverage", Value: resp.Leverage},
			{Key: "Amount", Value: resp.Amount},
			{Key: "Margin Amount", Value: resp.MarginAmount},
			{Key: "Available", Value: resp.Available},
			{Key: "Max Removable Margin", Value: resp.MaxRemovableMargin},
		})
	})
}
