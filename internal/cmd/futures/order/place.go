package order

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

// PlaceOptions captures the flag-bound state of `order place`.
type PlaceOptions struct {
	Symbol        string
	Limit         bool
	Market        bool
	Side          string
	Price         string
	Size          string
	ClientOrderID string
	TIF           string
	SLPrice       string
	SLTriggerBy   string
	TPPrice       string
	TPTriggerBy   string

	Factory *factory.Factory
}

// NewCmdPlace builds the `order place` cobra command.
//
// The cobra wiring stops here; runPlace below is pure orchestration.
//
// --limit / --market is the order-type pair. Cobra group annotations enforce:
//   - exactly one of --limit / --market must be set
//   - --price is only valid (and required) together with --limit
func NewCmdPlace(f *factory.Factory) *cobra.Command {
	opts := &PlaceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "place",
		Short: "Place a limit or market order",
		Example: "# Limit BUY: BTCUSDT at 70000 for 0.001\n" +
			"  100x futures order place --limit --symbol BTCUSDT --side buy --size 0.001 --price 70000\n\n" +
			"# Market SELL: BTCUSDT for 0.001\n" +
			"  100x futures order place --market --symbol BTCUSDT --side sell --size 0.001\n\n" +
			"# Limit BUY with attached SL 68000 and TP 76000\n" +
			"  100x futures order place --limit --symbol BTCUSDT --side buy --size 0.001 --price 70000 \\\n" +
			"      --sl-price 68000 --tp-price 76000",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().BoolVar(&opts.Limit, "limit", false, "Place a limit order (requires --price)")
	c.Flags().BoolVar(&opts.Market, "market", false, "Place a market order")
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "Trading pair, e.g. BTCUSDT")
	c.Flags().StringVar(&opts.Side, "side", "", "Order side: buy | sell")
	c.Flags().StringVar(&opts.Price, "price", "", "Limit price")
	c.Flags().StringVar(&opts.Size, "size", "", "Order quantity")
	c.Flags().StringVar(&opts.ClientOrderID, "client-order-id", "", "Client-supplied order ID")
	c.Flags().StringVar(&opts.TIF, "tif", "GTC", "Time in force for limit orders: GTC | IOC | FOK | POST_ONLY")
	c.Flags().StringVar(&opts.SLPrice, "sl-price", "", "Attach stop-loss at this price")
	c.Flags().StringVar(&opts.SLTriggerBy, "sl-trigger-by", "LAST", "Stop-loss trigger feed: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.TPPrice, "tp-price", "", "Attach take-profit at this price")
	c.Flags().StringVar(&opts.TPTriggerBy, "tp-trigger-by", "LAST", "Take-profit trigger feed: LAST | INDEX | MARK")
	_ = c.MarkFlagRequired("symbol")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("size")
	c.MarkFlagsMutuallyExclusive("limit", "market")
	c.MarkFlagsOneRequired("limit", "market")
	c.MarkFlagsRequiredTogether("limit", "price")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	_ = c.RegisterFlagCompletionFunc("side", complete.OrderSides)
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	_ = c.RegisterFlagCompletionFunc("tif", complete.TimeInForce)
	_ = c.RegisterFlagCompletionFunc("sl-trigger-by", complete.TriggerFeeds)
	_ = c.RegisterFlagCompletionFunc("tp-trigger-by", complete.TriggerFeeds)
	return c
}

// runPlace is the pure logic — no cobra dependency, callable from tests.
func runPlace(ctx context.Context, opts *PlaceOptions) error {
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"--price":    opts.Price,
		"--sl-price": opts.SLPrice,
		"--tp-price": opts.TPPrice,
	} {
		if err := clierr.PositiveNumber(name, value); err != nil {
			return err
		}
	}
	side, err := futures.ParseSide(opts.Side)
	if err != nil {
		return clierr.Usage(err)
	}
	tif, err := futures.ParseTIF(opts.TIF)
	if err != nil {
		return clierr.Usage(err)
	}
	slBy, err := futures.ParseStopTriggerType(opts.SLTriggerBy)
	if err != nil {
		return clierr.Usage(err)
	}
	tpBy, err := futures.ParseStopTriggerType(opts.TPTriggerBy)
	if err != nil {
		return clierr.Usage(err)
	}
	if !opts.Limit && !opts.Market {
		return clierr.Usagef("must set --limit or --market")
	}
	isStop := opts.SLPrice != "" || opts.TPPrice != ""
	f := opts.Factory
	ok, err := f.ConfirmDestructive(placeConfirmTitle(opts))
	if err != nil {
		return err
	}
	if !ok {
		return exit.NewCodedError(exit.Aborted, "cancelled", fmt.Errorf("cancelled by user"))
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	switch {
	case opts.Limit:
		resp, err := client.Order.LimitOrder(ctx, futures.LimitOrderReq{
			Market:              opts.Symbol,
			Side:                side,
			Price:               opts.Price,
			Quantity:            opts.Size,
			ClientOID:           opts.ClientOrderID,
			TIF:                 tif,
			IsStop:              isStop,
			StopLossPrice:       opts.SLPrice,
			StopLossPriceType:   slBy,
			TakeProfitPrice:     opts.TPPrice,
			TakeProfitPriceType: tpBy,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Symbol", Value: resp.Market},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			})
		})
	case opts.Market:
		resp, err := client.Order.MarketOrder(ctx, futures.MarketOrderReq{
			Market:              opts.Symbol,
			Side:                side,
			Quantity:            opts.Size,
			ClientOID:           opts.ClientOrderID,
			IsStop:              isStop,
			StopLossPrice:       opts.SLPrice,
			StopLossPriceType:   slBy,
			TakeProfitPrice:     opts.TPPrice,
			TakeProfitPriceType: tpBy,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Symbol", Value: resp.Market},
				{Key: "Status", Value: format.OrderStatus(f.IO, resp.Status)},
			})
		})
	}
	return nil
}

// placeConfirmTitle renders a one-line summary of the order for the
// destructive-op prompt: side, size, symbol, type/price, and any attached
// SL/TP triggers. The factory wrapper adds the [profile] prefix.
func placeConfirmTitle(opts *PlaceOptions) string {
	typeLabel := "MARKET"
	priceClause := ""
	if opts.Limit {
		typeLabel = "LIMIT"
		priceClause = fmt.Sprintf(" at %s", opts.Price)
	}
	var attached string
	switch {
	case opts.SLPrice != "" && opts.TPPrice != "":
		attached = fmt.Sprintf(" with SL %s, TP %s", opts.SLPrice, opts.TPPrice)
	case opts.SLPrice != "":
		attached = fmt.Sprintf(" with SL %s", opts.SLPrice)
	case opts.TPPrice != "":
		attached = fmt.Sprintf(" with TP %s", opts.TPPrice)
	}
	return fmt.Sprintf("Place %s %s %s %s%s%s?",
		typeLabel, strings.ToUpper(opts.Side), opts.Size, opts.Symbol, priceClause, attached)
}
