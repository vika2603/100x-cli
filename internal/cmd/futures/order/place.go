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
		Example: "# Place a BUY limit order on BTCUSDT at 70000 for size 0.001\n" +
			"  100x futures order place BTCUSDT --side buy --price 70000 --size 0.001\n\n" +
			"# Place a SELL market order on BTCUSDT for size 0.001\n" +
			"  100x futures order place BTCUSDT --type market --side sell --size 0.001\n\n" +
			"# Place a BUY limit order and attach SL 68000 plus TP 76000\n" +
			"  100x futures order place BTCUSDT --side buy --price 70000 --size 0.001 --sl 68000 --tp 76000",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Type, "type", "limit", "order type: limit | market")
	c.Flags().StringVar(&opts.Side, "side", "", "order side: buy | sell")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price; required for limit orders")
	c.Flags().StringVar(&opts.Size, "size", "", "order quantity")
	c.Flags().StringVar(&opts.ClientID, "client-id", "", "client-supplied order ID")
	c.Flags().StringVar(&opts.TIF, "tif", "GTC", "time in force for limit orders: GTC | IOC | FOK | POST_ONLY")
	c.Flags().StringVar(&opts.SL, "sl", "", "attach stop-loss at this price")
	c.Flags().StringVar(&opts.SLBy, "sl-by", "LAST", "stop-loss price feed: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.TP, "tp", "", "attach take-profit at this price")
	c.Flags().StringVar(&opts.TPBy, "tp-by", "LAST", "take-profit price feed: LAST | INDEX | MARK")
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
