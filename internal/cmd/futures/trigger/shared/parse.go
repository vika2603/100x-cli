// Package shared holds helpers used across multiple `trigger` verbs,
// principally the leg-preserve helpers used by `trigger attach order` and
// `trigger attach position`. The gateway endpoints take SL and TP together;
// preserving the untouched leg lives here so the CLI verbs do not bury that
// semantic in cobra glue.
package shared

import (
	"fmt"

	"github.com/vika2603/100x-cli/api/futures"
)

// ParseSide accepts the user-friendly forms.
func ParseSide(s string) (futures.Side, error) {
	switch s {
	case "buy", "BUY", "b":
		return futures.SideBuy, nil
	case "sell", "SELL", "s":
		return futures.SideSell, nil
	}
	return 0, fmt.Errorf("unknown side %q (want buy|sell)", s)
}

// ParsePriceType picks the trigger's price feed. The empty string
// defaults to the last-trade feed; any other unrecognised value is an
// error rather than a silent fallback.
func ParsePriceType(s string) (futures.StopTriggerType, error) {
	switch s {
	case "", "last", "Last":
		return futures.StopTriggerTypeLast, nil
	case "index", "Index":
		return futures.StopTriggerTypeIndex, nil
	case "mark", "Mark":
		return futures.StopTriggerTypeMark, nil
	}
	return 0, fmt.Errorf("unknown --price-type %q (want last|index|mark)", s)
}

// Leg is which leg an attach request is updating.
type Leg int

// Leg values.
const (
	LegSL Leg = iota
	LegTP
)

// ParseLeg accepts the user-typed labels.
func ParseLeg(s string) (Leg, error) {
	switch s {
	case "SL", "sl", "stop-loss":
		return LegSL, nil
	case "TP", "tp", "take-profit":
		return LegTP, nil
	}
	return 0, fmt.Errorf("unknown leg %q (want SL|TP)", s)
}
