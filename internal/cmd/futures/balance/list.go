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
	c.Flags().StringVar(&opts.Currency, "currency", "", "only show this asset, for example USDT")
	_ = c.RegisterFlagCompletionFunc("currency", complete.Assets)
	return c
}

// NewCmdBalances builds the `futures balances` shortcut for `futures balance list`.
func NewCmdBalances(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "balances",
		Short: "Show the current wallet snapshot",
		Long:  "Shortcut for `100x futures balance list`.",
		Example: "# Show balances for every asset in the wallet\n" +
			"  100x futures balances\n\n" +
			"# Filter the wallet snapshot down to USDT only\n" +
			"  100x futures balances --currency USDT",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Currency, "currency", "", "only show this asset, for example USDT")
	_ = c.RegisterFlagCompletionFunc("currency", complete.Assets)
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	resp, err := f.Client.Asset.AssetQuery(ctx, futures.AssetQueryReq{})
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
	return f.IO.Render(resp, func() error {
		if len(resp) == 0 {
			return f.IO.Emptyln("No balances found.")
		}
		rows := make([][]string, 0, len(resp))
		for _, b := range resp {
			rows = append(rows, []string{b.Asset, b.Available, b.Frozen, b.Margin, b.BalanceTotal, b.ProfitUnreal, b.Transfer})
		}
		return f.IO.Table([]output.Column{
			output.LCol("Asset"),
			output.RCol("Available"), output.RCol("Frozen"), output.RCol("Margin"),
			output.RCol("Total"), output.RCol("uPnL"), output.RCol("Transferable"),
		}, rows)
	})
}
