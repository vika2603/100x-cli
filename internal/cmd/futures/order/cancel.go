package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/exit"
)

// CancelOptions captures the flag-bound state of `order cancel`.
type CancelOptions struct {
	Symbol   string
	OrderIDs []string

	Factory *factory.Factory
}

// NewCmdCancel builds the `order cancel` cobra command.
func NewCmdCancel(f *factory.Factory) *cobra.Command {
	opts := &CancelOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel <symbol> <order-id> [<order-id>...]",
		Short: "Cancel one or more orders",
		Example: "# Cancel one open BTCUSDT order after a confirmation prompt\n" +
			"  100x futures order cancel BTCUSDT <order-id>\n\n" +
			"# Cancel two BTCUSDT orders immediately without the prompt\n" +
			"  100x futures order cancel BTCUSDT <id-1> <id-2> --yes",
		Args:              cobra.MinimumNArgs(2),
		ValidArgsFunction: complete.OpenOrderArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderIDs = args[1:]
			return runCancel(cmd.Context(), opts)
		},
	}
	return c
}

type cancelResult struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

func runCancel(ctx context.Context, opts *CancelOptions) error {
	f := opts.Factory
	for _, id := range opts.OrderIDs {
		if err := clierr.PositiveID("order-id", id); err != nil {
			return err
		}
	}
	if err := confirmCancelOrders(f, opts.Symbol, opts.OrderIDs); err != nil {
		return err
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	if len(opts.OrderIDs) == 1 {
		resp, err := client.Order.CancelOrder(ctx, futures.LimitOrderCancelReq{
			Market: opts.Symbol, OrderID: opts.OrderIDs[0],
		})
		if err != nil {
			return err
		}
		result := []cancelResult{{OrderID: opts.OrderIDs[0], Status: "CANCELED"}}
		return f.IO.Render(result, func() error {
			return f.IO.Resultln("Cancelled", resp.OrderID)
		})
	}
	resp, err := client.Order.LimitCancelOrderBatch(ctx, futures.LimitOrderCancelBatchReq{
		Market: opts.Symbol, OrderIDs: strings.Join(opts.OrderIDs, ","),
	})
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, id := range resp.OrderIDs {
		seen[id] = struct{}{}
	}
	results := make([]cancelResult, 0, len(opts.OrderIDs))
	missing := false
	for _, id := range opts.OrderIDs {
		status := "CANCELED"
		if _, ok := seen[id]; !ok {
			status = "UNKNOWN"
			missing = true
		}
		results = append(results, cancelResult{OrderID: id, Status: status})
	}
	if err := f.IO.Render(results, func() error {
		return f.IO.Resultln("Cancelled", len(resp.OrderIDs), "of", len(opts.OrderIDs), "orders in", opts.Symbol)
	}); err != nil {
		return err
	}
	if missing {
		return exit.NewCodedError(exit.Business, "business", fmt.Errorf("one or more orders were not confirmed canceled"))
	}
	return nil
}

func confirmCancelOrders(f *factory.Factory, symbol string, orderIDs []string) error {
	title := fmt.Sprintf("Cancel %d orders in %s?", len(orderIDs), symbol)
	if len(orderIDs) == 1 {
		title = fmt.Sprintf("Cancel order %s in %s?", orderIDs[0], symbol)
	}
	ok, err := f.ConfirmDestructive(title)
	if err != nil {
		return err
	}
	if !ok {
		return exit.NewCodedError(exit.Aborted, "cancelled", fmt.Errorf("cancelled by user"))
	}
	return nil
}

// CancelAllOptions captures the flag-bound state of `order cancel-all`.
type CancelAllOptions struct {
	Symbol string

	Factory *factory.Factory
}

// NewCmdCancelAll builds the `order cancel-all` cobra command.
func NewCmdCancelAll(f *factory.Factory) *cobra.Command {
	opts := &CancelAllOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel-all <symbol>",
		Short: "Cancel every open order in one market",
		Example: "# Cancel every open BTCUSDT order after a confirmation prompt\n" +
			"  100x futures order cancel-all BTCUSDT\n\n" +
			"# Cancel every open BTCUSDT order immediately without the prompt\n" +
			"  100x futures order cancel-all BTCUSDT --yes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.SymbolArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runCancelAll(cmd.Context(), opts)
		},
	}
	return c
}

func runCancelAll(ctx context.Context, opts *CancelAllOptions) error {
	f := opts.Factory
	ok, err := f.ConfirmDestructive(
		fmt.Sprintf("Cancel every open order in %s?", opts.Symbol))
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
	resp, err := client.Order.CancelAllOrder(ctx, futures.LimitOrderCancelAllReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Cancelled all open orders in", opts.Symbol)
	})
}
