package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/exit"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// CloseOptions captures the flag-bound state of `position close`.
type CloseOptions struct {
	Symbol        string
	PositionID    string
	Type          string
	Price         string
	Size          string
	ClientOrderID string

	Factory *factory.Factory
}

// NewCmdClose builds the `position close` cobra command.
func NewCmdClose(f *factory.Factory) *cobra.Command {
	opts := &CloseOptions{Factory: f}
	c := &cobra.Command{
		Use:   "close <symbol>",
		Short: "Close part or all of a position (limit or market)",
		Long: "Close part or all of a position.\n\n" +
			"Market close always closes the full position. Limit close requires both --price and --size.",
		Example: "# Close size 0.001 of one BTCUSDT position with a limit order at 80000\n" +
			"  100x futures position close BTCUSDT --position-id <position-id> --price 80000 --size 0.001\n\n" +
			"# Close the full BTCUSDT position at market without the prompt\n" +
			"  100x futures position close BTCUSDT --position-id <position-id> --type market --yes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runClose(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "Position ID; otherwise resolve from symbol")
	c.Flags().StringVar(&opts.Type, "type", "limit", "Execution type: limit | market")
	c.Flags().StringVar(&opts.Price, "price", "", "Limit price; required for limit orders")
	c.Flags().StringVar(&opts.Size, "size", "", "Quantity to close")
	c.Flags().StringVar(&opts.ClientOrderID, "client-order-id", "", "Client-supplied order ID")
	_ = c.RegisterFlagCompletionFunc("type", complete.OrderTypes)
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	_ = c.RegisterFlagCompletionFunc("position-id", complete.OpenPositionIDs)
	return c
}

func runClose(ctx context.Context, opts *CloseOptions) error {
	f := opts.Factory
	if err := clierr.PositiveID("position-id", opts.PositionID); opts.PositionID != "" && err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--price", opts.Price); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	if opts.Type == "market" && opts.Price != "" {
		return clierr.Usagef("--price is not allowed for market position close")
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	switch opts.Type {
	case "limit":
		if opts.Price == "" || opts.Size == "" {
			return clierr.Usagef("--price and --size are required for limit position close")
		}
		positionID, err := resolvePositionID(ctx, client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := client.Position.LimitClosePosition(ctx, futures.LimitClosePositionReq{
			Market: opts.Symbol, PositionID: positionID,
			Price: opts.Price, Quantity: opts.Size, ClientOID: opts.ClientOrderID,
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
		positionID, err := resolvePositionID(ctx, client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		ok, err := f.ConfirmDestructive(
			fmt.Sprintf("Close full position %s on %s at market?", positionID, opts.Symbol))
		if err != nil {
			return err
		}
		if !ok {
			return exit.ErrCancelled
		}
		if opts.Size != "" {
			f.IO.Println("warning: --size is ignored for market position close; server closes the full position")
		}
		resp, err := client.Position.MarketClosePosition(ctx, futures.MarketClosePositionReq{
			Market: opts.Symbol, PositionID: positionID,
			ClientOID: opts.ClientOrderID,
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
	return clierr.Usagef("unknown --type %q (want limit|market)", opts.Type)
}
