package complete

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/format"
	"github.com/vika2603/100x-cli/internal/session"
)

const completionTimeout = 2 * time.Second

// Assets completes wallet asset symbols from the configured private account.
func Assets(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, ctx, cancel, ok := privateClient(cmd)
	if !ok {
		return noFiles()
	}
	defer cancel()

	resp, err := client.Asset.AssetQuery(ctx, futures.AssetQueryReq{})
	if err != nil {
		return noFiles()
	}
	out := make([]string, 0, len(resp))
	for _, asset := range resp {
		out = append(out, asset.Asset)
	}
	return filtered(out, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// OpenOrderArgs completes a market symbol followed by an open order id.
func OpenOrderArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return Symbols(cmd, args, toComplete)
	}
	return completeOpenOrderIDs(cmd, args[0], toComplete)
}

// OpenOrderArg completes either a market symbol or one open order id.
func OpenOrderArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return Symbols(cmd, args, toComplete)
	case 1:
		return completeOpenOrderIDs(cmd, args[0], toComplete)
	default:
		return noFiles()
	}
}

// OpenOrderIDsFor returns a completion function for open order ids in symbol.
func OpenOrderIDsFor(symbol string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeOpenOrderIDs(cmd, symbol, toComplete)
	}
}

// OpenPositionArgs completes a market symbol followed by an open position id.
func OpenPositionArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return Symbols(cmd, args, toComplete)
	}
	return completeOpenPositionIDs(cmd, args[0], toComplete)
}

// OpenPositionArg completes either a market symbol or one open position id.
func OpenPositionArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return Symbols(cmd, args, toComplete)
	case 1:
		return completeOpenPositionIDs(cmd, args[0], toComplete)
	default:
		return noFiles()
	}
}

// OpenPositionIDs completes open position ids, optionally scoped by the first arg.
func OpenPositionIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	symbol := ""
	if len(args) > 0 {
		symbol = args[0]
	}
	return completeOpenPositionIDs(cmd, symbol, toComplete)
}

// OpenPositionIDsFor returns a completion function for open position ids in symbol.
func OpenPositionIDsFor(symbol string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeOpenPositionIDs(cmd, symbol, toComplete)
	}
}

// ActiveTriggerArgs completes a market symbol followed by an active trigger id.
func ActiveTriggerArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return Symbols(cmd, args, toComplete)
	}
	return completeActiveTriggerIDs(cmd, args[0], toComplete)
}

// ActiveTriggerArg completes either a market symbol or one active trigger id.
func ActiveTriggerArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return Symbols(cmd, args, toComplete)
	case 1:
		return completeActiveTriggerIDs(cmd, args[0], toComplete)
	default:
		return noFiles()
	}
}

// ActiveTriggerIDsFor returns a completion function for active trigger ids in symbol.
func ActiveTriggerIDsFor(symbol string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeActiveTriggerIDs(cmd, symbol, toComplete)
	}
}

func completeOpenOrderIDs(cmd *cobra.Command, symbol, toComplete string) ([]string, cobra.ShellCompDirective) {
	symbol = format.Market(symbol)
	client, ctx, cancel, ok := privateClient(cmd)
	if !ok {
		return noFiles()
	}
	defer cancel()

	resp, err := client.Order.PendingOrder(ctx, futures.PendingOrderReq{
		Market: symbol, Page: 1, PageSize: 50,
	})
	if err != nil {
		return noFiles()
	}
	out := make([]string, 0, len(resp.Records))
	for _, order := range resp.Records {
		out = append(out, strconv.FormatInt(order.OrderID, 10))
	}
	return filtered(out, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeOpenPositionIDs(cmd *cobra.Command, symbol, toComplete string) ([]string, cobra.ShellCompDirective) {
	symbol = format.Market(symbol)
	client, ctx, cancel, ok := privateClient(cmd)
	if !ok {
		return noFiles()
	}
	defer cancel()

	resp, err := client.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: symbol})
	if err != nil {
		return noFiles()
	}
	out := make([]string, 0, len(resp))
	for _, position := range resp {
		out = append(out, strconv.Itoa(position.PositionID))
	}
	return filtered(out, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeActiveTriggerIDs(cmd *cobra.Command, symbol, toComplete string) ([]string, cobra.ShellCompDirective) {
	symbol = format.Market(symbol)
	client, ctx, cancel, ok := privateClient(cmd)
	if !ok {
		return noFiles()
	}
	defer cancel()

	resp, err := client.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{
		Market: symbol, Page: 1, PageSize: 50,
	})
	if err != nil {
		return noFiles()
	}
	out := make([]string, 0, len(resp.Records))
	for _, trigger := range resp.Records {
		out = append(out, trigger.ContractOrderID)
	}
	return filtered(out, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// Symbols completes public futures market symbols.
func Symbols(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, ctx, cancel := publicClient(cmd)
	defer cancel()

	resp, err := client.Market.MarketList(ctx, futures.MarketListReq{})
	if err != nil {
		return noFiles()
	}
	out := make([]string, 0, len(resp))
	for _, market := range resp {
		value := market.Name
		if market.Stock != "" && market.Money != "" {
			value += "\t" + market.Stock + "/" + market.Money
		}
		out = append(out, value)
	}
	return filtered(out, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// SymbolArg completes a single public futures market symbol.
func SymbolArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return noFiles()
	}
	return Symbols(cmd, args, toComplete)
}

func publicClient(cmd *cobra.Command) (*futures.Client, context.Context, context.CancelFunc) {
	sess, _ := session.Load(session.LoadOptions{
		Timeout: completionTimeout,
		Public:  true,
	})
	ctx, cancel := completionContext(cmd)
	return sess.Client, ctx, cancel
}

func privateClient(cmd *cobra.Command) (*futures.Client, context.Context, context.CancelFunc, bool) {
	sess, err := session.Load(session.LoadOptions{
		RequestedProfile: profileFlag(cmd),
		Timeout:          completionTimeout,
	})
	if err != nil {
		return nil, nil, nil, false
	}
	ctx, cancel := completionContext(cmd)
	return sess.Client, ctx, cancel, true
}

func completionContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	base := context.Background()
	if cmd != nil {
		base = cmd.Context()
	}
	return context.WithTimeout(base, completionTimeout)
}

// profileFlag returns the explicit --profile flag value. The empty-string
// fallback to E100X_PROFILE / Config.Default is handled by config.Resolve
// inside session.Load.
func profileFlag(cmd *cobra.Command) string {
	if cmd != nil && cmd.Root() != nil {
		if flag := cmd.Root().PersistentFlags().Lookup("profile"); flag != nil {
			return flag.Value.String()
		}
	}
	return ""
}

func filtered(values []string, prefix string) []string {
	sort.Strings(values)
	if prefix == "" {
		return unique(values)
	}
	prefix = strings.ToLower(prefix)
	out := values[:0]
	for _, value := range values {
		if strings.HasPrefix(strings.ToLower(completionValue(value)), prefix) {
			out = append(out, value)
		}
	}
	return unique(out)
}

func completionValue(value string) string {
	value, _, _ = strings.Cut(value, "\t")
	return value
}

func unique(values []string) []string {
	out := values[:0]
	var last string
	for _, value := range values {
		if value == last {
			continue
		}
		out = append(out, value)
		last = value
	}
	return out
}

func noFiles() ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
