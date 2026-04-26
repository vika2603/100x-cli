package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/prompt"
)


// CancelOptions captures the flag-bound state of `order cancel`.
type CancelOptions struct {
	Market   string
	OrderIDs []string

	Factory *factory.Factory
}

// NewCmdCancel builds the `order cancel` cobra command.
func NewCmdCancel(f *factory.Factory) *cobra.Command {
	opts := &CancelOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel <order-id> [<order-id>...]",
		Short: "Cancel one or more orders",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderIDs = args
			return runCancel(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

func runCancel(ctx context.Context, opts *CancelOptions) error {
	f := opts.Factory
	if len(opts.OrderIDs) == 1 {
		resp, err := f.Client.Order.CancelOrder(ctx, futures.LimitOrderCancelReq{
			Market: opts.Market, OrderID: opts.OrderIDs[0],
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			f.IO.Println("cancelled", resp.OrderID)
			return nil
		})
	}
	resp, err := f.Client.Order.LimitCancelOrderBatch(ctx, futures.LimitOrderCancelBatchReq{
		Market: opts.Market, OrderIDs: strings.Join(opts.OrderIDs, ","),
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("cancelled", len(resp.OrderIDs), "orders in", opts.Market)
		return nil
	})
}

// CancelAllOptions captures the flag-bound state of `order cancel-all`.
type CancelAllOptions struct {
	Market string

	Factory *factory.Factory
}

// NewCmdCancelAll builds the `order cancel-all` cobra command.
func NewCmdCancelAll(f *factory.Factory) *cobra.Command {
	opts := &CancelAllOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel every open order in one market",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCancelAll(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

func runCancelAll(ctx context.Context, opts *CancelAllOptions) error {
	f := opts.Factory
	ok, err := prompt.ConfirmDestructive(
		fmt.Sprintf("Cancel every open order in %s?", opts.Market), f.Yes)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	resp, err := f.Client.Order.CancelAllOrder(ctx, futures.LimitOrderCancelAllReq{Market: opts.Market})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("cancelled all open orders in", opts.Market)
		return nil
	})
}
