package trigger

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
)

// PlaceOptions captures the flag-bound state of `trigger place`.
type PlaceOptions struct {
	Market       string
	Side         string
	StopPrice    string
	OrderPrice   string
	CurrentPrice string
	PriceType    string
	Quantity     string

	Factory *factory.Factory
}

// NewCmdPlace builds the `trigger place` cobra command.
func NewCmdPlace(f *factory.Factory) *cobra.Command {
	opts := &PlaceOptions{Factory: f}
	c := &cobra.Command{
		Use:   "place",
		Short: "Place a standalone trigger (condition order)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.Side, "side", "", "buy | sell")
	c.Flags().StringVar(&opts.StopPrice, "stop-price", "", "trigger price")
	c.Flags().StringVar(&opts.OrderPrice, "order-price", "", "limit price after trigger (omit = market)")
	c.Flags().StringVar(&opts.CurrentPrice, "current-price", "", "current-price snapshot (auto-fetched from ticker if omitted, matching --price-type)")
	c.Flags().StringVar(&opts.PriceType, "price-type", "last", "last | index | mark")
	c.Flags().StringVar(&opts.Quantity, "qty", "", "order quantity")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("stop-price")
	_ = c.MarkFlagRequired("qty")
	_ = c.RegisterFlagCompletionFunc("side", cobra.FixedCompletions([]string{"buy", "sell"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("price-type", cobra.FixedCompletions([]string{"last", "index", "mark"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runPlace(ctx context.Context, opts *PlaceOptions) error {
	side, err := shared.ParseSide(opts.Side)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.PriceType)
	if err != nil {
		return err
	}
	f := opts.Factory
	currentPrice := opts.CurrentPrice
	if currentPrice == "" {
		currentPrice, err = fetchCurrentPrice(ctx, f.Client, opts.Market, priceType)
		if err != nil {
			return err
		}
	}
	resp, err := f.Client.Order.StopOrder(ctx, futures.StopOrderReq{
		Market:        opts.Market,
		Side:          side,
		OrderPrice:    opts.OrderPrice,
		StopPrice:     opts.StopPrice,
		CutPrice:      currentPrice,
		StopPriceType: priceType,
		Quantity:      opts.Quantity,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("trigger placed in", opts.Market)
		return nil
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
