package position

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/complete"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/output"
)

// ListOptions captures the flag-bound state of `position list`.
type ListOptions struct {
	Symbol   string
	Page     int
	PageSize int

	Factory *factory.Factory
}

// NewCmdList builds the `position list` cobra command.
func NewCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List open positions",
		Example: "# List every open position in the account\n" +
			"  100x futures position list\n\n" +
			"# List open positions for BTCUSDT only\n" +
			"  100x futures position list --symbol BTCUSDT\n\n" +
			"# Extract position id, side, size, entry, upnl, and leverage as JSON\n" +
			"  100x --json futures position list --symbol BTCUSDT --jq 'map({position_id, side, volume, open_price, profit_unreal, leverage})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "only show positions for this symbol")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	return c
}

// NewCmdPositions builds the `futures positions` shortcut for `futures position list`.
func NewCmdPositions(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "positions",
		Short: "List open positions",
		Long:  "Shortcut for `100x futures position list`.",
		Example: "# List every open position in the account\n" +
			"  100x futures positions\n\n" +
			"# List open positions for BTCUSDT only\n" +
			"  100x futures positions --symbol BTCUSDT",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "only show positions for this symbol")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	resp, err := f.Client.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	if resp == nil {
		resp = []futures.PendingPositionDetail{}
	}
	return f.IO.Render(resp, func() error { return printOpen(f.IO, resp, opts.Symbol != "") })
}

// NewCmdHistory builds the `position history` cobra command.
func NewCmdHistory(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f, Page: 1, PageSize: 50}
	c := &cobra.Command{
		Use:   "history",
		Short: "List closed positions",
		Example: "# Show recently closed BTCUSDT positions with page size 20\n" +
			"  100x futures position history --symbol BTCUSDT --page-size 20\n\n" +
			"# Extract position id, side, open, close, pnl, and roe as JSON\n" +
			"  100x --json futures position history --symbol BTCUSDT --page-size 20 --jq 'map({position_id, side, open_price, close_price, profit_real, roe})'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := clierr.PositiveInt("--page", opts.Page); err != nil {
				return err
			}
			if err := clierr.PositiveInt("--page-size", opts.PageSize); err != nil {
				return err
			}
			resp, err := f.Client.Position.PositionHistory(cmd.Context(), futures.PositionHistoryReq{
				Market: opts.Symbol, Page: opts.Page, PageSize: opts.PageSize,
			})
			if err != nil {
				return err
			}
			records := resp.Records
			if records == nil {
				records = []futures.FinishedPositionDetail{}
			}
			return f.IO.Render(records, func() error { return printClosed(f.IO, records, opts.Symbol != "") })
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "only show closed positions for this symbol")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "items per page")
	_ = c.RegisterFlagCompletionFunc("symbol", complete.Symbols)
	return c
}

func printOpen(io *output.Renderer, rows []futures.PendingPositionDetail, symbolFiltered bool) error {
	if len(rows) == 0 {
		return io.Emptyln("No open positions found.")
	}
	cols := []output.Column{output.LCol("Position ID")}
	if !symbolFiltered {
		cols = append(cols, output.LCol("Symbol"))
	}
	cols = append(cols,
		output.LCol("Side"), output.RCol("Lev"),
		output.RCol("Size"), output.RCol("Entry"), output.RCol("Liq Price"),
		output.RCol("Margin"), output.RCol("uPnL"), output.RCol("ROE"),
		output.LCol("Opened"),
	)
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		row := []string{strconv.Itoa(p.PositionID)}
		if !symbolFiltered {
			row = append(row, p.Market)
		}
		row = append(row,
			format.Side(io, p.Side),
			p.Leverage+"x",
			p.Volume, p.OpenPrice, p.LiqPrice, p.MarginAmount, p.ProfitUnreal,
			format.Percent(p.Roe),
			format.UnixMillis(p.CreateTime),
		)
		out = append(out, row)
	}
	return io.Table(cols, out)
}

func printClosed(io *output.Renderer, rows []futures.FinishedPositionDetail, symbolFiltered bool) error {
	if len(rows) == 0 {
		return io.Emptyln("No closed positions found.")
	}
	cols := []output.Column{output.LCol("Position ID")}
	if !symbolFiltered {
		cols = append(cols, output.LCol("Symbol"))
	}
	cols = append(cols,
		output.LCol("Side"),
		output.RCol("Open"), output.RCol("Close"), output.RCol("Size"),
		output.RCol("PnL"), output.RCol("ROE"),
		output.LCol("Closed"),
	)
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		row := []string{strconv.Itoa(p.PositionID)}
		if !symbolFiltered {
			row = append(row, p.Market)
		}
		row = append(row,
			format.Side(io, p.Side),
			p.OpenPrice, p.ClosePrice, p.VolumeMax, p.ProfitReal,
			format.Percent(p.Roe),
			format.UnixMillis(p.UpdateTime),
		)
		out = append(out, row)
	}
	return io.Table(cols, out)
}
