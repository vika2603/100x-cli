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
		Long: "Inspect wallet balances and asset movement history.\n\n" +
			"Use `balance list` for the current account snapshot and `balance history` for paginated\n" +
			"asset changes such as deposits, withdrawals, and faucet activity.",
		Example: "# Show the current wallet balance snapshot for every asset\n" +
			"  100x futures balance list\n\n" +
			"# Review paginated asset history for USDT\n" +
			"  100x futures balance history --currency USDT --page-size 20",
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
		Long: "Show the current wallet snapshot.\n\n" +
			"The output includes available balance, frozen balance, margin usage, total balance,\n" +
			"unrealized PnL, and transferable amount for each asset. Use --currency to narrow the\n" +
			"view to one asset such as USDT.",
		Example: "# Show balances for every asset in the wallet\n" +
			"  100x futures balance list\n\n" +
			"# Filter the wallet snapshot down to USDT only\n" +
			"  100x futures balance list --currency USDT\n\n" +
			"# Extract asset, available, margin, total, and upnl as JSON\n" +
			"  100x --json futures balance list --jq 'map({asset, available, margin, balance_total, profit_unreal})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Currency, "currency", "", "only show this asset, for example USDT")
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
		Long: "List asset-change history for the current account.\n\n" +
			"Results are paginated. Use --currency to narrow to one asset and --type to filter by\n" +
			"business type such as deposit, withdraw, or faucet.",
		Example: "# Show recent USDT balance history with 20 items per page\n" +
			"  100x futures balance history --currency USDT --page-size 20\n\n" +
			"# Show only faucet records in USDT balance history\n" +
			"  100x futures balance history --currency USDT --type faucet\n\n" +
			"# Extract time, business type, and balance change as JSON\n" +
			"  100x --json futures balance history --currency USDT --jq 'map({time, business, change})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHistory(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Currency, "currency", "", "only show this asset (for example USDT)")
	c.Flags().StringVar(&opts.Type, "type", "", "business type: deposit | withdraw | faucet")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "items per page")
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
