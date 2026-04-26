package position

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/cmd/futures/style"
	"github.com/vika2603/100x-cli/internal/output"
)

// CloseOptions captures the flag-bound state of `position close`.
type CloseOptions struct {
	Market     string
	PositionID string
	Type       string
	Price      string
	Quantity   string
	ClientOID  string

	Factory *factory.Factory
}

// NewCmdClose builds the `position close` cobra command.
func NewCmdClose(f *factory.Factory) *cobra.Command {
	opts := &CloseOptions{Factory: f}
	c := &cobra.Command{
		Use:   "close",
		Short: "Close part or all of a position (limit or market)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClose(cmd.Context(), opts)
		},
	}
	c.Flags().StringVar(&opts.Market, "market", "", "instrument symbol")
	c.Flags().StringVar(&opts.PositionID, "position-id", "", "position id")
	c.Flags().StringVar(&opts.Type, "type", "limit", "limit | market")
	c.Flags().StringVar(&opts.Price, "price", "", "limit price (limit only)")
	c.Flags().StringVar(&opts.Quantity, "qty", "", "quantity to close")
	c.Flags().StringVar(&opts.ClientOID, "client-oid", "", "client order id")
	_ = c.MarkFlagRequired("market")
	_ = c.MarkFlagRequired("position-id")
	_ = c.MarkFlagRequired("qty")
	_ = c.RegisterFlagCompletionFunc("type", cobra.FixedCompletions([]string{"limit", "market"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func runClose(ctx context.Context, opts *CloseOptions) error {
	f := opts.Factory
	switch opts.Type {
	case "limit":
		resp, err := f.Client.Position.LimitClosePosition(ctx, futures.LimitClosePositionReq{
			Market: opts.Market, PositionID: opts.PositionID,
			Price: opts.Price, Quantity: opts.Quantity, ClientOID: opts.ClientOID,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Position ID", Value: opts.PositionID},
				{Key: "Market", Value: resp.Market},
				{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
				{Key: "Price", Value: resp.Price},
				{Key: "Qty", Value: resp.Volume},
			})
		})
	case "market":
		resp, err := f.Client.Position.MarketClosePosition(ctx, futures.MarketClosePositionReq{
			Market: opts.Market, PositionID: opts.PositionID,
			Quantity: opts.Quantity, ClientOID: opts.ClientOID,
		})
		if err != nil {
			return err
		}
		return f.IO.Render(resp, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Order ID", Value: strconv.FormatInt(resp.OrderID, 10)},
				{Key: "Position ID", Value: opts.PositionID},
				{Key: "Market", Value: resp.Market},
				{Key: "Status", Value: style.OrderStatus(f.IO, resp.Status)},
				{Key: "Price", Value: resp.Price},
				{Key: "Qty", Value: resp.Volume},
			})
		})
	}
	return fmt.Errorf("unknown --type %q (want limit|market)", opts.Type)
}
