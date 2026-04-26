// Package shared holds helpers used across `position` verbs.
package shared

import (
	"context"
	"fmt"

	"github.com/vika2603/100x-cli/api/futures"
)

// ParsePositionType accepts cross / isolated.
func ParsePositionType(s string) (futures.PositionType, error) {
	switch s {
	case "cross", "Cross":
		return futures.PositionTypeCross, nil
	case "isolated", "Isolated":
		return futures.PositionTypeIsolated, nil
	}
	return 0, fmt.Errorf("unknown position type %q (want cross|isolated)", s)
}

// MergedPreferenceInput describes a partial preference update; missing fields
// are filled in from the gateway's current state.
type MergedPreferenceInput struct {
	Market       string
	Leverage     string // empty = preserve
	PositionType string // empty = preserve; else "cross"|"isolated"
}

// BuildAdjustMarketPreferenceReq performs the read-modify-send compensation:
// the gateway's POST /setting/preference takes leverage AND position_type
// together, so a partial CLI update reads current values first and merges.
func BuildAdjustMarketPreferenceReq(ctx context.Context, c *futures.Client, in MergedPreferenceInput) (futures.AdjustMarketPreferenceReq, error) {
	out := futures.AdjustMarketPreferenceReq{Market: in.Market}
	if in.Leverage == "" || in.PositionType == "" {
		cur, err := c.Setting.MarketPreference(ctx, futures.MarketPreferenceReq{Market: in.Market})
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

// ParseMarginAction accepts add / remove.
func ParseMarginAction(s string) (futures.MarginAction, error) {
	switch s {
	case "add":
		return futures.MarginActionAdd, nil
	case "remove", "sub", "subtract":
		return futures.MarginActionRemove, nil
	}
	return 0, fmt.Errorf("unknown margin action %q (want add|remove)", s)
}
