package position

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// ListOptions captures the flag-bound state of `position list`.
type ListOptions struct {
	Market   string
	History  bool
	Page     int
	PageSize int

	Factory *factory.Factory
}

// NewCmdList builds the `position list` cobra command.
func NewCmdList(f *factory.Factory) *cobra.Command {
	opts := &ListOptions{Factory: f}
	c := &cobra.Command{
		Use:   "list",
		Short: "List open or closed positions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "filter by market")
	c.Flags().BoolVar(&opts.History, "history", false, "list closed positions instead of open ones")
	c.Flags().IntVar(&opts.Page, "page", 1, "page number")
	c.Flags().IntVar(&opts.PageSize, "page-size", 50, "page size")
	return c
}

func runList(ctx context.Context, opts *ListOptions) error {
	f := opts.Factory
	if opts.History {
		resp, err := f.Client.Position.PositionHistory(ctx, futures.PositionHistoryReq{
			Market: opts.Market, Page: opts.Page, PageSize: opts.PageSize,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error { return printClosed(f.IO, resp.Records) })
	}
	resp, err := f.Client.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: opts.Market})
	if err != nil {
		return err
	}
	return f.IO.Render(resp, func() error { return printOpen(f.IO, resp) })
}

func printOpen(io *output.Renderer, rows []futures.PendingPositionDetail) error {
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		out = append(out, []string{
			strconv.Itoa(p.PositionID), p.Market, style.Side(io, p.Side),
			p.Volume, p.OpenPrice, p.LiqPrice, p.MarginAmount, p.ProfitUnreal, p.Roe,
		})
	}
	return io.Table([]string{"ID", "Market", "Side", "Qty", "Entry", "Liq Price", "Margin", "uPnL", "ROE"}, out)
}

func printClosed(io *output.Renderer, rows []futures.FinishedPositionDetail) error {
	out := make([][]string, 0, len(rows))
	for _, p := range rows {
		out = append(out, []string{
			strconv.Itoa(p.PositionID), p.Market, style.Side(io, p.Side),
			p.OpenPrice, p.ClosePrice, p.VolumeMax, p.ProfitReal, p.Roe,
		})
	}
	return io.Table([]string{"ID", "Market", "Side", "Open", "Close", "Qty", "PnL", "ROE"}, out)
}
