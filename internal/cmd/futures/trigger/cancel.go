package trigger

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/exit"
	"github.com/vika2603/100x-cli/internal/prompt"
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
		Args:  cobra.MinimumNArgs(2),
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
	if f.DryRun {
		f.IO.Println("dry-run: cancel triggers", strings.Join(opts.OrderIDs, ","), "in", opts.Symbol)
		return nil
	}
	if err := confirmCancelTriggers(f, opts.Symbol, opts.OrderIDs); err != nil {
		return err
	}
	ids := make([]int64, 0, len(opts.OrderIDs))
	for _, id := range opts.OrderIDs {
		resp, err := f.Client.Order.CancelStopOrder(ctx, futures.StopOrderCancelReq{
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
	ok, err := prompt.ConfirmDestructive(title, f.Yes)
	if err != nil {
		return err
	}
	if !ok {
		return exit.NewCodedError(exit.Aborted, "cancelled", fmt.Errorf("cancelled by user"))
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Symbol = args[0]
			return runCancelAll(cmd.Context(), opts)
		},
	}
	return c
}

func runCancelAll(ctx context.Context, opts *CancelAllOptions) error {
	f := opts.Factory
	if f.DryRun {
		f.IO.Println("dry-run: cancel all active triggers in", opts.Symbol)
		return nil
	}
	ok, err := prompt.ConfirmDestructive(
		fmt.Sprintf("Cancel every active trigger in %s?", opts.Symbol), f.Yes)
	if err != nil {
		return err
	}
	if !ok {
		return exit.NewCodedError(exit.Aborted, "cancelled", fmt.Errorf("cancelled by user"))
	}
	resp, err := f.Client.Order.CancelAllStopOrder(ctx, futures.StopOrderCancelAllReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		return f.IO.Resultln("Cancelled all triggers in", opts.Symbol)
	})
}
