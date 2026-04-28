package balance

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/output"
)

// ListOptions captures the state of `balance list`.
type ListOptions struct {
	Currency string

	Factory *factory.Factory
}

func newCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show the current wallet snapshot",
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
	c.Flags().StringVar(&opts.Currency, "currency", "", "Only show this asset, for example USDT")
	_ = c.RegisterFlagCompletionFunc("currency", complete.Assets)
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	client, err := f.Futures()
	if err != nil {
		return err
	}
	resp, err := client.Asset.AssetQuery(ctx, futures.AssetQueryReq{})
	if err != nil {
		return err
	}
	if resp == nil {
		resp = []futures.AssetDetailItem{}
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
	currencyFiltered := opts.Currency != ""
	return f.IO.Render(resp, func() error {
		if len(resp) == 0 {
			return f.IO.Emptyln("No balances found.")
		}
		cols := []output.Column{}
		if !currencyFiltered {
			cols = append(cols, output.LCol("Asset"))
		}
		cols = append(cols,
			output.RCol("Available"), output.RCol("Frozen"), output.RCol("Margin"),
			output.RCol("Total"), output.RCol("uPnL"), output.RCol("Bonus"),
			output.RCol("Transferable"),
		)
		rows := make([][]string, 0, len(resp))
		for _, b := range resp {
			row := []string{}
			if !currencyFiltered {
				row = append(row, b.Asset)
			}
			row = append(row,
				b.Available, b.Frozen, b.Margin, b.BalanceTotal,
				b.ProfitUnreal, b.Bonus, b.Transfer,
			)
			rows = append(rows, row)
		}
		return f.IO.Table(cols, rows)
	})
}
