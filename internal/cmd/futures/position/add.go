package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// AddOptions captures the flag-bound state of `position add`.
type AddOptions struct {
	Symbol     string
	PositionID string
	Type       string
	Price      string
	Size       string

	Factory *factory.Factory
}

// NewCmdAdd builds the `position add` cobra command.
func NewCmdAdd(f *factory.Factory) *cobra.Command {
	opts := &AddOptions{Factory: f}
	c := &cobra.Command{
		Use:   "add <symbol>",
		Short: "Top up an existing position (limit or market)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runAdd(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "position id")
	c.Flags().StringVar(&opts.Type, "type", "limit", "limit | market")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price (limit only)")
	c.Flags().StringVar(&opts.Size, "size", "", "size to add")
	_ = c.MarkFlagRequired("size")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"limit", "market"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runAdd(ctx context.Context, opts *AddOptions) error {
	f := opts.Factory
	switch opts.Type {
	case "limit":
		if opts.Price == "" {
			return fmt.Errorf("--price is required for limit position add")
		}
		if f.DryRun {
			f.IO.Println("dry-run: limit position add", opts.Symbol, "position", opts.PositionID, "price", opts.Price, "size", opts.Size)
			return nil
		}
		positionID, err := resolvePositionID(ctx, f.Client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := f.Client.Position.LimitAddPosition(ctx, futures.LimitAddPositionReq{
			Market: opts.Symbol, PositionID: positionID,
			Price: opts.Price, Quantity: opts.Size,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Position ID", Value: positionID},
				{Key: "Symbol", Value: resp.Market},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
				{Key: "Price", Value: resp.Price},
				{Key: "Size", Value: resp.Volume},
			})
		})
	case "market":
		if f.DryRun {
			f.IO.Println("dry-run: market position add", opts.Symbol, "position", opts.PositionID, "size", opts.Size)
			return nil
		}
		positionID, err := resolvePositionID(ctx, f.Client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := f.Client.Position.MarketAddPosition(ctx, futures.MarketAddPositionReq{
			Market: opts.Symbol, PositionID: positionID,
			Quantity: opts.Size,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Position ID", Value: positionID},
				{Key: "Symbol", Value: resp.Market},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
				{Key: "Price", Value: resp.Price},
				{Key: "Size", Value: resp.Volume},
			})
		})
	}
	return fmt.Errorf("unknown --type %q (want limit|market)", opts.Type)
}
