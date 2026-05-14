package position

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/exit"
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
		Short: "Add to an existing position (limit or market)",
		Example: "# Add size 0.001 to one BTCUSDT position with a limit order at 70000\n" +
			"  100x futures position add BTCUSDT --position-id <position-id> --price 70000 --size 0.001\n\n" +
			"# Add size 0.001 to one BTCUSDT position at market\n" +
			"  100x futures position add BTCUSDT --position-id <position-id> --type market --size 0.001",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runAdd(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "Position ID; otherwise resolve from symbol")
	c.Flags().StringVar(&opts.Type, "type", "limit", "Execution type: limit | market")
	c.Flags().StringVar(&opts.Price, "price", "", "Limit price; required for limit orders")
	c.Flags().StringVar(&opts.Size, "size", "", "Quantity to add")
	_ = c.MarkFlagRequired("size")
	_ = c.RegisterFlagCompletionFunc("type", complete.OrderTypes)
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	_ = c.RegisterFlagCompletionFunc("position-id", complete.OpenPositionIDs)
	return c
}

func runAdd(ctx context.Context, opts *AddOptions) error {
	f := opts.Factory
	if err := clierr.PositiveID("position-id", opts.PositionID); opts.PositionID != "" && err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--price", opts.Price); err != nil {
		return err
	}
	if opts.Type != "limit" && opts.Type != "market" {
		return clierr.Usagef("unknown --type %q (want limit|market)", opts.Type)
	}
	if opts.Type == "market" && opts.Price != "" {
		return clierr.Usagef("--price is not allowed for market position add")
	}
	if opts.Type == "limit" && opts.Price == "" {
		return clierr.Usagef("--price is required for limit position add")
	}
	ok, err := f.ConfirmDestructive(addConfirmTitle(opts))
	if err != nil {
		return err
	}
	if !ok {
		return exit.ErrCancelled
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	switch opts.Type {
	case "limit":
		positionID, err := resolvePositionID(ctx, client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := client.Position.LimitAddPosition(ctx, futures.LimitAddPositionReq{
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
		positionID, err := resolvePositionID(ctx, client, opts.Symbol, opts.PositionID)
		if err != nil {
			return err
		}
		resp, err := client.Position.MarketAddPosition(ctx, futures.MarketAddPositionReq{
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
	return nil
}

// addConfirmTitle renders a one-line summary of the add operation for the
// destructive-op prompt: size, symbol, type, and optional limit price.
func addConfirmTitle(opts *AddOptions) string {
	typeLabel := strings.ToUpper(opts.Type)
	priceClause := ""
	if opts.Type == "limit" {
		priceClause = fmt.Sprintf(" at %s", opts.Price)
	}
	return fmt.Sprintf("Add %s to %s position (%s%s)?", opts.Size, opts.Symbol, typeLabel, priceClause)
}
