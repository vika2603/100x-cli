// Package shared holds helpers used across `position` verbs.
package shared

import (
	"context"
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
)

// ParsePositionType accepts cross / isolated.
func ParsePositionType(s string) (futures.PositionType, error) {
	switch strings.ToUpper(s) {
	case "CROSS":
		return futures.PositionTypeCross, nil
	case "ISOLATED":
		return futures.PositionTypeIsolated, nil
	}
	return 0, clierr.Usagef("unknown mode %q (want ISOLATED|CROSS)", s)
}

// MergedPreferenceInput describes a partial preference update; missing fields
// are filled in from the gateway's current state.
type MergedPreferenceInput struct {
	Symbol       string
	Leverage     string // empty = preserve
	PositionType string // empty = preserve; else "CROSS"|"ISOLATED"
}

// BuildAdjustMarketPreferenceReq performs the read-modify-send compensation:
// the gateway's POST /setting/preference takes leverage AND position_type
// together, so a partial CLI update reads current values first and merges.
func BuildAdjustMarketPreferenceReq(ctx context.Context, c *futures.Client, in MergedPreferenceInput) (futures.AdjustMarketPreferenceReq, error) {
	out := futures.AdjustMarketPreferenceReq{Market: in.Symbol}
	if in.Leverage == "" || in.PositionType == "" {
		cur, err := c.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: in.Symbol})
		if err != nil {
			return out, err
		}
		if in.Leverage == "" {
			out.Leverage = cur.Leverage
		}
		if in.PositionType == "" {
			out.PositionType = cur.PositionType
		}
	}
	if in.Leverage != "" {
		out.Leverage = in.Leverage
	}
	if in.PositionType != "" {
		pt, err := ParsePositionType(in.PositionType)
		if err != nil {
			return out, err
		}
		out.PositionType = pt
	}
	return out, nil
}
