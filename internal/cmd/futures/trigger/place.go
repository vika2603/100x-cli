package trigger

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
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
		Short: "Place a standalone trigger (condition order)",
		Long: "Place a standalone trigger (condition order).\n\n" +
			"The CLI fetches the current price automatically when needed for gateway validation.",
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
	c.Flags().StringVar(&opts.Side, "side", "", "triggered order side: buy | sell")
	c.Flags().StringVar(&opts.Size, "size", "", "triggered order quantity")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "price that activates this trigger")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "trigger feed: LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.LimitPrice, "limit-price", "", "limit price after the trigger fires; omit for market execution")
	c.Flags().StringVar(&opts.CurrentPrice, "current-price", "", "override the current price snapshot for testing")
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
	if err := clierr.PositiveNumber("--size", opts.Size); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--trigger-price", opts.TriggerPrice); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--limit-price", opts.LimitPrice); err != nil {
		return err
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
	var priceType futures.StopTriggerType
	switch strings.ToUpper(opts.TriggerBy) {
	case "", "LAST":
		priceType = futures.StopTriggerTypeLast
	case "INDEX":
		priceType = futures.StopTriggerTypeIndex
	case "MARK":
		priceType = futures.StopTriggerTypeMark
	default:
		return clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", opts.TriggerBy)
	}
	f := opts.Factory
	currentPrice := opts.CurrentPrice
	if currentPrice == "" {
		fetched, err := fetchCurrentPrice(ctx, f.Client, opts.Symbol, priceType)
		if err != nil {
			return err
		}
		currentPrice = fetched
	}
	resp, err := f.Client.Order.StopOrder(ctx, futures.StopOrderReq{
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
