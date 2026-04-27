package balance

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

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
			"business type such as deposit, withdraw, faucet, fee, or trade. The gateway accepts\n" +
			"more values than this list; an unknown --type returns no rows rather than an error.",
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
	c.Flags().StringVar(&opts.Currency, "currency", "", "Only show this asset (for example USDT)")
	c.Flags().StringVar(&opts.Type, "type", "", "Business type, for example deposit | withdraw | faucet | fee | trade")
	c.Flags().IntVar(&opts.Page, "page", 1, "Page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "Items per page")
	_ = c.RegisterFlagCompletionFunc("currency", complete.Assets)
	_ = c.RegisterFlagCompletionFunc("type", complete.BalanceEventTypes)
	return c
}

func runHistory(ctx context.Context, opts *HistoryOptions) error {
	f := opts.Factory
	if err := clierr.PositiveInt("--page", opts.Page); err != nil {
		return err
	}
	if err := clierr.PositiveInt("--page-size", opts.PageSize); err != nil {
		return err
	}
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
		if len(records) == 0 {
			return f.IO.Emptyln("No balance history found.")
		}
		rows := make([][]string, 0, len(records))
		for _, r := range records {
			rows = append(rows, []string{format.UnixMillis(r.Time), r.Asset, format.Enum(r.Business), r.Change})
		}
		return f.IO.Table([]output.Column{
			output.LCol("Time"), output.LCol("Asset"), output.LCol("Business"),
			output.RCol("Change"),
		}, rows)
	})
}
