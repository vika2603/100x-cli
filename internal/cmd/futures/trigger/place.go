package trigger

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/wire"
)

// PlaceOptions captures the flag-bound state of `trigger place`.
type PlaceOptions struct {
	Symbol       string
	Side         string
	TriggerPrice string
	LimitPrice   string
	CurrentPrice string
	TriggerBy    string
	Size         string

	Factory *factory.Factory
}

// NewCmdPlace builds the `trigger place` cobra command.
func NewCmdPlace(f *factory.Factory) *cobra.Command {
	opts := &PlaceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "place <symbol>",
		Short: "Place a standalone conditional order",
		Long: "Place a standalone conditional order that fires when the trigger price is reached.\n\n" +
			"Pass --limit-price to submit a limit order on trigger; omit it to execute at market.\n" +
			"Use --trigger-by to choose which feed (LAST, INDEX, or MARK) the trigger watches.",
		Example: "# Place a BUY trigger on BTCUSDT that executes at market when 65000 is reached\n" +
			"  100x futures trigger place BTCUSDT --side buy --trigger-price 65000 --size 0.001\n\n" +
			"# Place a SELL trigger that submits a limit order at 81950 when 82000 is reached\n" +
			"  100x futures trigger place BTCUSDT --side sell --trigger-price 82000 --limit-price 81950 --size 0.001",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Side, "side", "", "Triggered order side: buy | sell")
	c.Flags().StringVar(&opts.Size, "size", "", "Triggered order quantity")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "Price that activates this trigger")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Trigger feed: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.LimitPrice, "limit-price", "", "Limit price after the trigger fires; omit for market execution")
	c.Flags().StringVar(&opts.CurrentPrice, "current-price", "", "Override the current price snapshot for testing")
	_ = c.Flags().MarkHidden("current-price")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("size")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("side", complete.OrderSides)
	_ = c.RegisterFlagCompletionFunc("size", complete.OrderSizes)
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	return c
}

func runPlace(ctx context.Context, opts *PlaceOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--trigger-price", opts.TriggerPrice); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--limit-price", opts.LimitPrice); err != nil {
		return err
	}
	side, err := futures.ParseSide(opts.Side)
	if err != nil {
		return clierr.Usage(err)
	}
	priceType, err := futures.ParseStopTriggerType(opts.TriggerBy)
	if err != nil {
		return clierr.Usage(err)
	}
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	currentPrice := opts.CurrentPrice
	if currentPrice == "" {
		fetched, err := fetchCurrentPrice(ctx, client, opts.Symbol, priceType)
		if err != nil {
			return err
		}
		currentPrice = fetched
	}
	resp, err := client.Order.StopOrder(ctx, futures.StopOrderReq{
		Market:        opts.Symbol,
		Side:          side,
		OrderPrice:    opts.LimitPrice,
		StopPrice:     opts.TriggerPrice,
		CutPrice:      currentPrice,
		StopPriceType: priceType,
		Quantity:      opts.Size,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Created trigger on", opts.Symbol)
	})
}

// fetchCurrentPrice pulls the ticker snapshot and returns the price that
// matches the chosen trigger feed (last / index / mark). The gateway
// validates the standalone-trigger request against this current-price
// snapshot, rejecting with code=20021 if it is missing or stale.
func fetchCurrentPrice(ctx context.Context, c *futures.Client, market string, t futures.StopTriggerType) (string, error) {
	state, err := c.Market.MarketState(ctx, futures.MarketStateReq{Market: market})
	if err != nil {
		return "", err
	}
	switch t {
	case futures.StopTriggerTypeIndex:
		return state.IndexPrice, nil
	case futures.StopTriggerTypeMark:
		return state.SignPrice, nil
	default:
		return state.Last, nil
	}
}
