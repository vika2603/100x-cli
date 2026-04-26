package order

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/order/shared"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// PlaceOptions captures the flag-bound state of `order place`.
type PlaceOptions struct {
	Type      string
	Market    string
	Side      string
	Price     string
	Quantity  string
	ClientOID string
	TIF       string

	Factory *factory.Factory
}

// NewCmdPlace builds the `order place` cobra command.
//
// The cobra wiring stops here; runPlace below is pure orchestration.
func NewCmdPlace(f *factory.Factory) *cobra.Command {
	opts := &PlaceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "place",
		Short: "Place a limit or market order",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Type, "type", "limit", "limit | market")
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol (e.g. BTCUSDT)")
	c.Flags().StringVar(&opts.Side, "side", "", "buy | sell")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price (limit only)")
	c.Flags().StringVar(&opts.Quantity, "qty", "", "order quantity")
	c.Flags().StringVar(&opts.ClientOID, "client-oid", "", "optional client-side order id")
	c.Flags().StringVar(&opts.TIF, "tif", "GTC", "GTC | FOK | IOC | PostOnly")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("qty")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"limit", "market"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("side", cobra.FixedCompletions([]string{"buy", "sell"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("tif", cobra.FixedCompletions([]string{"GTC", "FOK", "IOC", "PostOnly"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

// runPlace is the pure logic — no cobra dependency, callable from tests.
func runPlace(ctx context.Context, opts *PlaceOptions) error {
	side, err := shared.ParseSide(opts.Side)
	if err != nil {
		return err
	}
	tif, err := shared.ParseTIF(opts.TIF)
	if err != nil {
		return err
	}
	f := opts.Factory
	switch opts.Type {
	case "limit":
		if f.DryRun {
			f.IO.Println("dry-run: limit", opts.Market, opts.Side, opts.Price, "qty", opts.Quantity)
			return nil
		}
		resp, err := f.Client.Order.LimitOrder(ctx, futures.LimitOrderReq{
			Market:    opts.Market,
			Side:      side,
			Price:     opts.Price,
			Quantity:  opts.Quantity,
			ClientOID: opts.ClientOID,
			TIF:       tif,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Market", Value: opts.Market},
				{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
			})
		})
	case "market":
		if f.DryRun {
			f.IO.Println("dry-run: market", opts.Market, opts.Side, "qty", opts.Quantity)
			return nil
		}
		resp, err := f.Client.Order.MarketOrder(ctx, futures.MarketOrderReq{
			Market:    opts.Market,
			Side:      side,
			Quantity:  opts.Quantity,
			ClientOID: opts.ClientOID,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Market", Value: opts.Market},
				{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
			})
		})
	}
	return fmt.Errorf("unknown --type %q (want limit|market)", opts.Type)
}
