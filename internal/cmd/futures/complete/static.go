// Package complete contains static shell completions for futures commands.
package complete

import "github.com/spf13/cobra"

func NoFiles(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func BalanceEventTypes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("deposit", "withdraw", "faucet")
}

func KlineIntervals(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values(
		"1m", "5m", "10m", "15m", "30m",
		"1h", "2h", "4h", "6h", "12h",
		"1d", "1w", "1M",
		"1min", "5min", "10min", "15min", "30min",
		"1hour", "2hour", "4hour", "6hour", "12hour",
		"1day", "1week", "1month",
	)
}

func MarginModes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("ISOLATED", "CROSS")
}

func OrderSides(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("buy", "sell")
}

func OrderTypes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("limit", "market")
}

func OrderSizes(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("0.001", "0.01", "0.1", "1", "10", "100")
}

func TimeExpressions(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("now", "now-15m", "now-1h", "now-4h", "now-24h", "now-7d")
}

func TimeInForce(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("GTC", "IOC", "FOK", "POST_ONLY")
}

func TriggerFeeds(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("LAST", "INDEX", "MARK")
}

func TriggerLegs(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return values("SL", "TP")
}

func values(items ...string) ([]string, cobra.ShellCompDirective) {
	return items, cobra.ShellCompDirectiveNoFileComp
}
