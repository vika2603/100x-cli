package trigger

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/exit"
)

// CancelOptions captures the flag-bound state of `trigger cancel`.
type CancelOptions struct {
	Symbol   string
	OrderIDs []string

	Factory *factory.Factory
}

// NewCmdCancel builds the `trigger cancel` cobra command.
func NewCmdCancel(f *factory.Factory) *cobra.Command {
	opts := &CancelOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel <symbol> <trigger-id> [<trigger-id>...]",
		Short: "Cancel one or more pending triggers",
		Example: "# Cancel one pending BTCUSDT trigger after a confirmation prompt\n" +
			"  100x futures trigger cancel BTCUSDT <trigger-id>\n\n" +
			"# Cancel two pending BTCUSDT triggers immediately without the prompt\n" +
			"  100x futures trigger cancel BTCUSDT <id-1> <id-2> --yes",
		Args:              cobra.MinimumNArgs(2),
		ValidArgsFunction: complete.ActiveTriggerArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			opts.OrderIDs = args[1:]
			return runCancel(cmd.Context(), opts)
		},
	}
	return c
}

func runCancel(ctx context.Context, opts *CancelOptions) error {
	f := opts.Factory
	for _, id := range opts.OrderIDs {
		if err := clierr.PositiveID("trigger-id", id); err != nil {
			return err
		}
	}
	if err := confirmCancelTriggers(f, opts.Symbol, opts.OrderIDs); err != nil {
		return err
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(opts.OrderIDs))
	for _, id := range opts.OrderIDs {
		resp, err := client.Order.CancelStopOrder(ctx, futures.StopOrderCancelReq{
			Market: opts.Symbol, OrderID: id,
		})
		if err != nil {
			return err
		}
		ids = append(ids, resp.OrderID)
	}
	return f.IO.Render(ids, func() error {
		return f.IO.Resultln("Cancelled", len(ids), "triggers in", opts.Symbol)
	})
}

func confirmCancelTriggers(f *factory.Factory, symbol string, triggerIDs []string) error {
	title := fmt.Sprintf("Cancel %d triggers in %s?", len(triggerIDs), symbol)
	if len(triggerIDs) == 1 {
		title = fmt.Sprintf("Cancel trigger %s in %s?", triggerIDs[0], symbol)
	}
	ok, err := f.ConfirmDestructive(title)
	if err != nil {
		return err
	}
	if !ok {
		return exit.ErrCancelled
	}
	return nil
}

// CancelAllOptions captures the flag-bound state of `trigger cancel-all`.
type CancelAllOptions struct {
	Symbol string

	Factory *factory.Factory
}

// NewCmdCancelAll builds the `trigger cancel-all` cobra command.
func NewCmdCancelAll(f *factory.Factory) *cobra.Command {
	opts := &CancelAllOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel-all <symbol>",
		Short: "Cancel every active trigger in one market",
		Example: "# Cancel every active BTCUSDT trigger after a confirmation prompt\n" +
			"  100x futures trigger cancel-all BTCUSDT\n\n" +
			"# Cancel every active BTCUSDT trigger immediately without the prompt\n" +
			"  100x futures trigger cancel-all BTCUSDT --yes",
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
		fmt.Sprintf("Cancel every active trigger in %s?", opts.Symbol))
	if err != nil {
		return err
	}
	if !ok {
		return exit.ErrCancelled
	}
	client, err := f.Futures()
	if err != nil {
		return err
	}
	resp, err := client.Order.CancelAllStopOrder(ctx, futures.StopOrderCancelAllReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Cancelled all triggers in", opts.Symbol)
	})
}
