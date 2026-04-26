package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/exit"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/prompt"
)

// CloseOptions captures the flag-bound state of `position close`.
type CloseOptions struct {
	Symbol     string
	PositionID string
	Type       string
	Price      string
	Size       string
	ClientID   string

	Factory *factory.Factory
}

// NewCmdClose builds the `position close` cobra command.
func NewCmdClose(f *factory.Factory) *cobra.Command {
	opts := &CloseOptions{Factory: f}
	c := &cobra.Command{
		Use:   "close <symbol>",
		Short: "Close part or all of a position (limit or market)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runClose(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "position id")
	c.Flags().StringVar(&opts.Type, "type", "limit", "limit | market")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price (limit only)")
	c.Flags().StringVar(&opts.Size, "size", "", "size to close")
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "client order id")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"limit", "market"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runClose(ctx context.Context, opts *CloseOptions) error {
	f := opts.Factory
	switch opts.Type {
	case "limit":
		if opts.Price == "" || opts.Size == "" {
			return fmt.Errorf("--price and --size are required for limit position close")
		}
		if f.DryRun {
			f.IO.Println("dry-run: limit position close", opts.Symbol, "position", opts.PositionID, "price", opts.Price, "size", opts.Size)
			return nil
		}
		positionID, err := resolvePositionID(ctx, f.Client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := f.Client.Position.LimitClosePosition(ctx, futures.LimitClosePositionReq{
			Market: opts.Symbol, PositionID: positionID,
			Price: opts.Price, Quantity: opts.Size, ClientOID: opts.ClientID,
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
			f.IO.Println("dry-run: market position close", opts.Symbol, "position", opts.PositionID)
			return nil
		}
		positionID, err := resolvePositionID(ctx, f.Client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		ok, err := prompt.ConfirmDestructive(
			fmt.Sprintf("Close full position %s on %s at market?", positionID, opts.Symbol), f.Yes)
		if err != nil {
			return err
		}
		if !ok {
			return exit.NewCodedError(exit.Aborted, "cancelled", fmt.Errorf("cancelled by user"))
		}
		if opts.Size != "" {
			f.IO.Println("warning: --size is ignored for market position close; server closes the full position")
		}
		resp, err := f.Client.Position.MarketClosePosition(ctx, futures.MarketClosePositionReq{
			Market: opts.Symbol, PositionID: positionID,
			ClientOID: opts.ClientID,
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
