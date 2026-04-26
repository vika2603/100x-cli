package trigger

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
	"github.com/vika2603/100x-cli/internal/output"
)

// EditOptions captures the flag-bound state of `trigger edit`.
type EditOptions struct {
	Market    string
	OrderID   string
	StopPrice string
	PriceType string

	Factory *factory.Factory
}

// NewCmdEdit builds the `trigger edit` cobra command.
func NewCmdEdit(f *factory.Factory) *cobra.Command {
	opts := &EditOptions{Factory: f}
	c := &cobra.Command{
		Use:   "edit <trigger-id>",
		Short: "Modify a pending trigger (attached SL/TP only)",
		Long: "Modify a pending trigger.\n\n" +
			"Only attached SL/TP triggers (created via `trigger attach`) can be edited.\n" +
			"Standalone triggers (created via `trigger place`) cannot be edited; cancel\n" +
			"and resubmit instead.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderID = args[0]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.StopPrice, "stop-price", "", "new trigger price")
	c.Flags().StringVar(&opts.PriceType, "price-type", "last", "last | index | mark")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("stop-price")
	_ = c.RegisterFlagCompletionFunc("price-type", cobra.FixedCompletions([]string{"last", "index", "mark"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	priceType, err := shared.ParsePriceType(opts.PriceType)
	if err != nil {
		return err
	}
	f := opts.Factory
	resp, err := f.Client.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
		Market:        opts.Market,
		StopOrderID:   opts.OrderID,
		StopPrice:     opts.StopPrice,
		StopPriceType: priceType,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Market", Value: opts.Market},
			{Key: "Stop Price", Value: opts.StopPrice},
			{Key: "Price Type", Value: opts.PriceType},
		})
	})
}
