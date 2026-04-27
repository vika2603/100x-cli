// Package shared holds helpers used across multiple `trigger` verbs,
// principally the leg-preserve helpers used by `trigger attach order` and
// `trigger attach position`. The gateway endpoints take SL and TP together;
// preserving the untouched leg lives here so the CLI verbs do not bury that
// semantic in cobra glue.
package shared

import (
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
)

// ParseSide accepts the user-friendly forms.
func ParseSide(s string) (futures.Side, error) {
	switch strings.ToUpper(s) {
	case "BUY", "B":
		return futures.SideBuy, nil
	case "SELL", "S":
		return futures.SideSell, nil
	}
	return 0, clierr.Usagef("unknown side %q (want buy|sell)", s)
}

// ParsePriceType picks the trigger's price feed. The empty string
// defaults to the last-trade feed; any other unrecognised value is an
// error rather than a silent fallback.
func ParsePriceType(s string) (futures.StopTriggerType, error) {
	switch strings.ToUpper(s) {
	case "", "LAST":
		return futures.StopTriggerTypeLast, nil
	case "INDEX":
		return futures.StopTriggerTypeIndex, nil
	case "MARK":
		return futures.StopTriggerTypeMark, nil
	}
	return 0, clierr.Usagef("unknown trigger price type %q (want LAST|INDEX|MARK)", s)
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
	switch strings.ToUpper(s) {
	case "SL", "STOP-LOSS":
		return LegSL, nil
	case "TP", "TAKE-PROFIT":
		return LegTP, nil
	}
	return 0, clierr.Usagef("unknown leg %q (want SL|TP)", s)
}
