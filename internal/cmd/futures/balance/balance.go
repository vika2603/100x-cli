// Package balance wires the `100x futures balance` verbs.
package balance

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/format"
)

// NewCmdBalance returns the `balance` group.
func NewCmdBalance(f *factory.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "balance",
		Short: "Wallet balance and asset history",
	}
	c.AddCommand(newCmdList(f), newCmdHistory(f))
	return c
}

// ListOptions captures the state of `balance list`.
type ListOptions struct {
	Currency string

	Factory *factory.Factory
}

func newCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "list",
		Short: "Show the current wallet snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Currency, "currency", "", "client-side currency filter")
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	resp, err := f.Client.Asset.AssetQuery(ctx, futures.AssetQueryReq{})
	if err != nil {
		return err
	}
	if opts.Currency != "" {
		currency := strings.ToUpper(opts.Currency)
		filtered := resp[:0]
		for _, b := range resp {
			if strings.ToUpper(b.Asset) == currency {
				filtered = append(filtered, b)
			}
		}
		resp = filtered
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
	Currency string
	Type     string
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
	c.Flags().StringVar(&opts.Currency, "currency", "", "filter by currency (e.g. USDT)")
	c.Flags().StringVar(&opts.Type, "type", "", "filter by business type (deposit | withdraw | faucet)")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "page size")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"deposit", "withdraw", "faucet"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runHistory(ctx context.Context, opts *HistoryOptions) error {
	f := opts.Factory
	resp, err := f.Client.Asset.AssetHistory(ctx, futures.AssetHistoryReq{
		Asset: strings.ToUpper(opts.Currency), Business: opts.Type, Page: opts.Page, PageSize: opts.PageSize,
	})
	if err != nil {
		return err
	}
	records := resp.Records
	if records == nil {
		records = []futures.AssetHistoryItem{}
	}
	return f.IO.Render(records, func() error {
		rows := make([][]string, 0, len(records))
		for _, r := range records {
			rows = append(rows, []string{format.UnixMillis(r.Time), r.Asset, format.Enum(r.Business), r.Change})
		}
		return f.IO.Table([]string{"Time", "Asset", "Business", "Change"}, rows)
	})
}
