package position

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
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
		Use:   "list",
		Short: "List open positions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "filter by symbol")
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	resp, err := f.Client.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: opts.Symbol})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error { return printOpen(f.IO, resp) })
}

// NewCmdHistory builds the `position history` cobra command.
func NewCmdHistory(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f, Page: 1, PageSize: 50}
	c := &cobra.Command{
		Use:   "history",
		Short: "List closed positions",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			return f.IO.Render(records, func() error { return printClosed(f.IO, records) })
		},
	}
	c.Flags().StringVar(&opts.Symbol, "symbol", "", "filter by symbol")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 20, "page size")
	return c
}

func printOpen(io *output.Renderer, rows []futures.PendingPositionDetail) error {
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		out = append(out, []string{
			strconv.Itoa(p.PositionID), p.Market, format.Side(io, p.Side),
			p.Volume, p.OpenPrice, p.LiqPrice, p.MarginAmount, p.ProfitUnreal, p.Roe,
		})
	}
	return io.Table([]string{"Position ID", "Symbol", "Side", "Size", "Entry", "Liq Price", "Margin", "uPnL", "ROE"}, out)
}

func printClosed(io *output.Renderer, rows []futures.FinishedPositionDetail) error {
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		out = append(out, []string{
			strconv.Itoa(p.PositionID), p.Market, format.Side(io, p.Side),
			p.OpenPrice, p.ClosePrice, p.VolumeMax, p.ProfitReal, p.Roe,
		})
	}
	return io.Table([]string{"Position ID", "Symbol", "Side", "Open", "Close", "Size", "PnL", "ROE"}, out)
}
