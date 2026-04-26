// Package balance wires the `100x futures balance` verbs.
package balance

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
)

// NewCmdBalance returns the `balance` group.
func NewCmdBalance(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "balance",
		Short: "Wallet balance and asset history",
	}
	c.AddCommand(newCmdGet(f), newCmdHistory(f))
	return c
}

// GetOptions captures the (empty) state of `balance get`.
type GetOptions struct {
	Factory *factory.Factory
}

func newCmdGet(f *factory.Factory) *cobra.Command {
	opts := &GetOptions{Factory: f}
	c := &cobra.Command{
		Use:   "get",
		Short: "Show the current wallet snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGet(cmd.Context(), opts)
		},
	}
	return c
}

func runGet(ctx context.Context, opts *GetOptions) error {
	f := opts.Factory
	resp, err := f.Client.Asset.AssetQuery(ctx, futures.AssetQueryReq{})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error {
		rows := make([][]string, 0, len(resp))
		for _, b := range resp {
			rows = append(rows, []string{b.Asset, b.Available, b.Frozen, b.Margin, b.BalanceTotal, b.ProfitUnreal, b.Transfer})
		}
		return f.IO.Table([]string{"Asset", "Available", "Frozen", "Margin", "Total", "uPnL", "Transferable"}, rows)
	})
}

// HistoryOptions captures the flag-bound state of `balance history`.
type HistoryOptions struct {
	Asset    string
	Business string
	Page     int
	PageSize int

	Factory *factory.Factory
}

func newCmdHistory(f *factory.Factory) *cobra.Command {
	opts := &HistoryOptions{Factory: f}
	c := &cobra.Command{
		Use:   "history",
		Short: "List asset-change history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHistory(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Asset, "asset", "", "filter by asset (e.g. USDT)")
	c.Flags().StringVar(&opts.Business, "business", "", "filter by business (deposit | withdraw | faucet)")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 50, "page size")
	_ = c.RegisterFlagCompletionFunc("business", cobra.FixedCompletions([]string{"deposit", "withdraw", "faucet"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runHistory(ctx context.Context, opts *HistoryOptions) error {
	f := opts.Factory
	resp, err := f.Client.Asset.AssetHistory(ctx, futures.AssetHistoryReq{
		Asset: opts.Asset, Business: opts.Business, Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, nil)
}
