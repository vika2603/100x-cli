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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runPlace(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Side, "side", "", "buy | sell")
	c.Flags().StringVar(&opts.Size, "size", "", "order size")
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "LAST | INDEX | MARK")
	c.Flags().StringVar(&opts.LimitPrice, "limit-price", "", "limit price after trigger (omit = market)")
	c.Flags().StringVar(&opts.CurrentPrice, "current-price", "", "current-price snapshot")
	_ = c.Flags().MarkHidden("current-price")
	_ = c.MarkFlagRequired("side")
	_ = c.MarkFlagRequired("size")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("side", cobra.FixedCompletions([]string{"buy", "sell"}, cobra.ShellCompDirectiveNoFileComp))
	_ = c.RegisterFlagCompletionFunc("trigger-by", cobra.FixedCompletions([]string{"LAST", "INDEX", "MARK"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runPlace(ctx context.Context, opts *PlaceOptions) error {
	side, err := shared.ParseSide(opts.Side)
	if err != nil {
		return err
	}
	priceType, err := shared.ParsePriceType(opts.TriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	if f.DryRun {
		f.IO.Println("dry-run: place trigger", opts.Symbol, opts.Side, "trigger", opts.TriggerPrice, "size", opts.Size)
		return nil
	}
	currentPrice := opts.CurrentPrice
	if currentPrice == "" {
		currentPrice, err = fetchCurrentPrice(ctx, f.Client, opts.Symbol, priceType)
		if err != nil {
			return err
		}
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
