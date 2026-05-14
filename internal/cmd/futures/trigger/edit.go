package trigger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/cmd/futures/protection"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/wire"
)

// EditOptions captures the flag-bound state of `trigger edit`.
type EditOptions struct {
	Symbol       string
	OrderID      string
	TriggerPrice string
	TriggerBy    string

	Factory *factory.Factory
}

// NewCmdEdit builds the `trigger edit` cobra command.
func NewCmdEdit(f *factory.Factory) *cobra.Command {
	opts := &EditOptions{Factory: f}
	c := &cobra.Command{
		Use:   "edit <symbol> <trigger-id>",
		Short: "Modify a pending trigger (attached SL/TP only)",
		Long: "Modify a pending trigger.\n\n" +
			"Only attached SL/TP triggers (created via `trigger attach`) can be edited.\n" +
			"Standalone triggers (created via `trigger place`) cannot be edited; cancel\n" +
			"and resubmit instead.",
		Example: "# Change one attached BTCUSDT trigger to price 69000\n" +
			"  100x futures trigger edit BTCUSDT <trigger-id> --trigger-price 69000\n\n" +
			"# Change the trigger price to 69000 and use MARK as the trigger feed\n" +
			"  100x futures trigger edit BTCUSDT <trigger-id> --trigger-price 69000 --trigger-by MARK",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.ActiveTriggerArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "New trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "Trigger feed: LAST | INDEX | MARK")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("trigger-by", complete.TriggerFeeds)
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	opts.Symbol = wire.Market(opts.Symbol)
	if err := clierr.PositiveID("trigger-id", opts.OrderID); err != nil {
		return err
	}
	if err := clierr.PositiveNumber("--trigger-price", opts.TriggerPrice); err != nil {
		return err
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
	attached, err := protection.IsAttached(ctx, client, opts.Symbol, opts.OrderID)
	if err != nil {
		return err
	}
	if !attached {
		return fmt.Errorf("trigger %s cannot be edited: it is either a standalone trigger or no longer pending. Standalone triggers must be cancelled and re-placed; run `100x futures trigger list` to see editable triggers", opts.OrderID)
	}
	resp, err := client.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
		Market:        opts.Symbol,
		StopOrderID:   opts.OrderID,
		StopPrice:     opts.TriggerPrice,
		StopPriceType: priceType,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		// StopOrderEditResp only carries the trigger id; user supplied the
		// symbol on the command line, so don't echo a client-derived value.
		return f.IO.Object([]output.KV{
			{Key: "Trigger ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Trigger Price", Value: opts.TriggerPrice},
			{Key: "Trigger By", Value: format.Enum(opts.TriggerBy)},
		})
	})
}
