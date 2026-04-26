package order

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/order/shared"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// PlaceOptions captures the flag-bound state of `order place`.
type PlaceOptions struct {
	Type     string
	Symbol   string
	Side     string
	Price    string
	Size     string
	ClientID string
	TIF      string
	SL       string
	SLBy     string
	TP       string
	TPBy     string

	Factory *factory.Factory
}

// NewCmdPlace builds the `order place` cobra command.
//
// The cobra wiring stops here; runPlace below is pure orchestration.
func NewCmdPlace(f *factory.Factory) *cobra.Command {
	opts := &PlaceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "place <symbol>",
		Short: "Place a limit or market order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Type, "type", "limit", "limit | market")
	c.Flags().StringVar(&opts.Side, "side", "", "buy | sell")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price (limit only)")
	c.Flags().StringVar(&opts.Size, "size", "", "order size")
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "optional client-side order id")
	c.Flags().StringVar(&opts.TIF, "tif", "GTC", "GTC | IOC | FOK | POST_ONLY")
	c.Flags().StringVar(&opts.SL, "sl", "", "stop-loss trigger price")
	c.Flags().StringVar(&opts.SLBy, "sl-by", "LAST", "LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.TP, "tp", "", "take-profit trigger price")
	c.Flags().StringVar(&opts.TPBy, "tp-by", "LAST", "LAST | INDEX | MARK")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("size")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"limit", "market"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("side", cobra.FixedCompletions([]string{"buy", "sell"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("tif", cobra.FixedCompletions([]string{"GTC", "FOK", "IOC", "POST_ONLY"}, cobra.ShellCompDirectiveNoFileComp))
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
	slBy, err := shared.ParseStopTriggerType(opts.SLBy)
	if err != nil {
		return err
	}
	tpBy, err := shared.ParseStopTriggerType(opts.TPBy)
	if err != nil {
		return err
	}
	isStop := opts.SL != "" || opts.TP != ""
	f := opts.Factory
	switch opts.Type {
	case "limit":
		if opts.Price == "" {
			return fmt.Errorf("--price is required for limit orders")
		}
		if f.DryRun {
			f.IO.Println("dry-run: limit", opts.Symbol, opts.Side, opts.Price, "size", opts.Size)
			return nil
		}
		resp, err := f.Client.Order.LimitOrder(ctx, futures.LimitOrderReq{
			Market:        opts.Symbol,
			Side:          side,
			Price:         opts.Price,
			Quantity:      opts.Size,
			ClientOID:     opts.ClientID,
			TIF:           tif,
			IsStop:        isStop,
			StopLossPrice: opts.SL, StopLossPriceType: slBy,
			TakeProfitPrice: opts.TP, TakeProfitPriceType: tpBy,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Symbol", Value: opts.Symbol},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			})
		})
	case "market":
		if f.DryRun {
			f.IO.Println("dry-run: market", opts.Symbol, opts.Side, "size", opts.Size)
			return nil
		}
		resp, err := f.Client.Order.MarketOrder(ctx, futures.MarketOrderReq{
			Market:        opts.Symbol,
			Side:          side,
			Quantity:      opts.Size,
			ClientOID:     opts.ClientID,
			IsStop:        isStop,
			StopLossPrice: opts.SL, StopLossPriceType: slBy,
			TakeProfitPrice: opts.TP, TakeProfitPriceType: tpBy,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Symbol", Value: opts.Symbol},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			})
		})
	}
	return fmt.Errorf("unknown --type %q (want limit|market)", opts.Type)
}
