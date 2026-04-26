// Package shared holds helpers used across multiple `order` verbs.
package shared

import (
	"fmt"
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
)

// ParseSide accepts the user-friendly forms and yields a typed Side.
func ParseSide(s string) (futures.Side, error) {
	switch strings.ToUpper(s) {
	case "BUY", "B":
		return futures.SideBuy, nil
	case "SELL", "S":
		return futures.SideSell, nil
	}
	return 0, fmt.Errorf("unknown side %q (want buy|sell)", s)
}

// ParseTIF accepts the labels users type and yields a typed TIF. The
// empty string defaults to GTC; any other unrecognised value is an
// error rather than a silent fallback.
func ParseTIF(s string) (futures.TIF, error) {
	switch strings.ToUpper(s) {
	case "", "GTC":
		return futures.TIFGTC, nil
	case "FOK":
		return futures.TIFFOK, nil
	case "IOC":
		return futures.TIFIOC, nil
	case "POST_ONLY", "POSTONLY", "PO":
		return futures.TIFPostOnly, nil
	}
	return 0, fmt.Errorf("unknown --tif %q (want GTC|FOK|IOC|POST_ONLY)", s)
}

// ParseStopTriggerType accepts the trigger feed names used by order SL/TP.
func ParseStopTriggerType(s string) (futures.StopTriggerType, error) {
	switch strings.ToUpper(s) {
	case "", "LAST":
		return futures.StopTriggerTypeLast, nil
	case "INDEX":
		return futures.StopTriggerTypeIndex, nil
	case "MARK":
		return futures.StopTriggerTypeMark, nil
	}
	return 0, fmt.Errorf("unknown trigger price type %q (want LAST|INDEX|MARK)", s)
}
