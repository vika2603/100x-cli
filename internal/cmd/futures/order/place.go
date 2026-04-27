package order

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
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
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
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
	_ = c.RegisterFlagCompletionFunc("type", complete.OrderTypes)
	_ = c.RegisterFlagCompletionFunc("side", complete.OrderSides)
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	_ = c.RegisterFlagCompletionFunc("tif", complete.TimeInForce)
	_ = c.RegisterFlagCompletionFunc("sl-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("tp-by", complete.TriggerFeeds)
	return c
}

// runPlace is the pure logic — no cobra dependency, callable from tests.
func runPlace(ctx context.Context, opts *PlaceOptions) error {
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"--price": opts.Price,
		"--sl":    opts.SL,
		"--tp":    opts.TP,
	} {
		if err := clierr.PositiveNumber(name, value); err != nil {
			return err
		}
	}
	var side futures.Side
	switch strings.ToUpper(opts.Side) {
	case "BUY", "B":
		side = futures.SideBuy
	case "SELL", "S":
		side = futures.SideSell
	default:
		return clierr.Usagef("unknown side %q (want buy|sell)", opts.Side)
	}
	var tif futures.TIF
	switch strings.ToUpper(opts.TIF) {
	case "", "GTC":
		tif = futures.TIFGTC
	case "FOK":
		tif = futures.TIFFOK
	case "IOC":
		tif = futures.TIFIOC
	case "POST_ONLY", "POSTONLY", "PO":
		tif = futures.TIFPostOnly
	default:
		return clierr.Usagef("unknown --tif %q (want GTC|FOK|IOC|POST_ONLY)", opts.TIF)
	}
	var slBy futures.StopTriggerType
	switch strings.ToUpper(opts.SLBy) {
	case "", "LAST":
		slBy = futures.StopTriggerTypeLast
	case "INDEX":
		slBy = futures.StopTriggerTypeIndex
	case "MARK":
		slBy = futures.StopTriggerTypeMark
	default:
		return clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", opts.SLBy)
	}
	var tpBy futures.StopTriggerType
	switch strings.ToUpper(opts.TPBy) {
	case "", "LAST":
		tpBy = futures.StopTriggerTypeLast
	case "INDEX":
		tpBy = futures.StopTriggerTypeIndex
	case "MARK":
		tpBy = futures.StopTriggerTypeMark
	default:
		return clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", opts.TPBy)
	}
	isStop := opts.SL != "" || opts.TP != ""
	f := opts.Factory
	if opts.Type == "market" && opts.Price != "" {
		return clierr.Usagef("--price is not allowed for market orders")
	}
	switch opts.Type {
	case "limit":
		if opts.Price == "" {
			return clierr.Usagef("--price is required for limit orders")
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
	return clierr.Usagef("unknown --type %q (want limit|market)", opts.Type)
}
