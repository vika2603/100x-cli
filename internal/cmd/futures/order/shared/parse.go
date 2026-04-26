// Package shared holds helpers used across multiple `order` verbs.
package shared

import (
	"fmt"

	"github.com/vika2603/100x-cli/api/futures"
)

// ParseSide accepts the user-friendly forms and yields a typed Side.
func ParseSide(s string) (futures.Side, error) {
	switch s {
	case "buy", "BUY", "b":
		return futures.SideBuy, nil
	case "sell", "SELL", "s":
		return futures.SideSell, nil
	}
	return 0, fmt.Errorf("unknown side %q (want buy|sell)", s)
}

// ParseTIF accepts the labels users type and yields a typed TIF. The
// empty string defaults to GTC; any other unrecognised value is an
// error rather than a silent fallback.
func ParseTIF(s string) (futures.TIF, error) {
	switch s {
	case "", "GTC":
		return futures.TIFGTC, nil
	case "FOK":
		return futures.TIFFOK, nil
	case "IOC":
		return futures.TIFIOC, nil
	case "PostOnly", "PO":
		return futures.TIFPostOnly, nil
	}
	return 0, fmt.Errorf("unknown --tif %q (want GTC|FOK|IOC|PostOnly)", s)
}
