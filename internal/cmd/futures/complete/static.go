// Package complete contains static shell completions for futures commands.
package complete

import "github.com/spf13/cobra"

// NoFiles disables filesystem completion.
func NoFiles(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// BalanceEventTypes completes balance history business types. The gateway
// accepts more values than this; the list covers the common ones surfaced
// in --help.
func BalanceEventTypes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("deposit", "withdraw", "faucet", "fee", "trade")
}

// KlineIntervalAliases is the canonical short-form interval set advertised
// in error messages and help text.
var KlineIntervalAliases = []string{
	"1m", "5m", "10m", "15m", "30m",
	"1h", "2h", "4h", "6h", "12h",
	"1d", "1w", "1M",
}

// KlineIntervalNatives is the gateway-native form of each alias. parseInterval
// pass-through accepts these too, but they are not the canonical CLI form.
var KlineIntervalNatives = []string{
	"1min", "5min", "10min", "15min", "30min",
	"1hour", "2hour", "4hour", "6hour", "12hour",
	"1day", "1week", "1month",
}

// KlineIntervals completes supported market kline intervals.
func KlineIntervals(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	all := append([]string{}, KlineIntervalAliases...)
	all = append(all, KlineIntervalNatives...)
	return values(all...)
}

// MarginModes completes supported position margin modes.
func MarginModes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("ISOLATED", "CROSS")
}

// OrderSides completes order side values.
func OrderSides(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("buy", "sell")
}

// OrderTypes completes order type values.
func OrderTypes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("limit", "market")
}

// OrderSizes completes common order size examples.
func OrderSizes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("0.001", "0.01", "0.1", "1", "10", "100")
}

// TimeExpressions completes common relative time expressions.
func TimeExpressions(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("now", "now-15m", "now-1h", "now-4h", "now-24h", "now-7d")
}

// TimeInForce completes limit order time-in-force values.
func TimeInForce(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("GTC", "IOC", "FOK", "POST_ONLY")
}

// TriggerFeeds completes supported trigger price feeds.
func TriggerFeeds(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("LAST", "INDEX", "MARK")
}

// TriggerLegs completes stop-loss and take-profit labels.
func TriggerLegs(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("SL", "TP")
}

func values(items ...string) ([]string, cobra.ShellCompDirective) {
	return items, cobra.ShellCompDirectiveNoFileComp
}
