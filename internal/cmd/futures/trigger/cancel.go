package trigger

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/prompt"
)

// CancelOptions captures the flag-bound state of `trigger cancel`.
type CancelOptions struct {
	Market  string
	OrderID string

	Factory *factory.Factory
}

// NewCmdCancel builds the `trigger cancel` cobra command.
func NewCmdCancel(f *factory.Factory) *cobra.Command {
	opts := &CancelOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel <trigger-id>",
		Short: "Cancel one pending trigger",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.OrderID = args[0]
			return runCancel(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	_ = c.MarkFlagRequired("market")
	return c
}

func runCancel(ctx context.Context, opts *CancelOptions) error {
	f := opts.Factory
	resp, err := f.Client.Order.CancelStopOrder(ctx, futures.StopOrderCancelReq{
		Market: opts.Market, OrderID: opts.OrderID,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("cancelled trigger", resp.OrderID)
		return nil
	})
}

// CancelAllOptions captures the flag-bound state of `trigger cancel-all`.
type CancelAllOptions struct {
	Market string

	Factory *factory.Factory
}

// NewCmdCancelAll builds the `trigger cancel-all` cobra command.
func NewCmdCancelAll(f *factory.Factory) *cobra.Command {
	opts := &CancelAllOptions{Factory: f}
	c := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel every active trigger in one market",
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
		fmt.Sprintf("Cancel every active trigger in %s?", opts.Market), f.Yes)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	resp, err := f.Client.Order.CancelAllStopOrder(ctx, futures.StopOrderCancelAllReq{Market: opts.Market})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		f.IO.Println("cancelled all triggers in", opts.Market)
		return nil
	})
}
