package trigger

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/trigger/shared"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
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
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderID = args[1]
			return runEdit(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.TriggerPrice, "trigger-price", "", "new trigger price")
	c.Flags().StringVar(&opts.TriggerBy, "trigger-by", "LAST", "trigger feed: LAST | INDEX | MARK")
	_ = c.MarkFlagRequired("trigger-price")
	_ = c.RegisterFlagCompletionFunc("trigger-by", cobra.FixedCompletions([]string{"LAST", "INDEX", "MARK"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	priceType, err := shared.ParsePriceType(opts.TriggerBy)
	if err != nil {
		return err
	}
	f := opts.Factory
	resp, err := f.Client.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
		Market:        opts.Symbol,
		StopOrderID:   opts.OrderID,
		StopPrice:     opts.TriggerPrice,
		StopPriceType: priceType,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Trigger ID", Value: strconv.FormatInt(resp.OrderID, 10)},
			{Key: "Symbol", Value: opts.Symbol},
			{Key: "Trigger Price", Value: opts.TriggerPrice},
			{Key: "Trigger By", Value: format.Enum(opts.TriggerBy)},
		})
	})
}
