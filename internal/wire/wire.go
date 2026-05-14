// Package wire normalises user-facing input into the gateway's wire format.
//
// CLI users routinely type symbols as "btcusdt", "BTC-USDT", or "BTCUSDT".
// The gateway only accepts the all-uppercase, hyphen-free form. Each verb
// that takes a --symbol flag or positional <symbol> argument routes that
// value through wire.Market before placing it on a request struct.
package wire

import "strings"

// Market converts a user-supplied market symbol into the gateway wire format:
// uppercase, with hyphens removed. Empty input is returned unchanged.
func Market(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(strings.ReplaceAll(s, "-", ""))
}
