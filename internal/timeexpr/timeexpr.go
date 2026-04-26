// Package timeexpr parses CLI time expressions into gateway Unix seconds.
package timeexpr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Help describes the accepted CLI time formats for --since / --until.
const Help = "Unix seconds, Unix milliseconds, YYYY-MM-DD HH:MM:SS, now, now-24h, or now+24h; empty --until defaults to now when --since is set"

var layouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

// Parse accepts Unix seconds, Unix milliseconds, local/RFC3339 timestamps, or
// relative expressions such as now-24h and now+7d.
func Parse(value, name string) (int, error) {
	return parseAt(value, name, time.Now())
}

// ResolveRange parses a since/until pair against one shared `now`.
// When --since is set and --until is omitted, until defaults to the current
// time so relative windows like `--since now-24h` are ergonomic.
func ResolveRange(since, until string) (int, int, error) {
	now := time.Now()
	start, err := parseAt(since, "since", now)
	if err != nil {
		return 0, 0, err
	}
	if strings.TrimSpace(until) == "" && strings.TrimSpace(since) != "" {
		return start, int(now.Unix()), nil
	}
	end, err := parseAt(until, "until", now)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func parseAt(value, name string, now time.Time) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if ts, ok, err := parseNow(value, now); ok || err != nil {
		return ts, err
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		if n > 1_000_000_000_000 {
			n /= 1000
		}
		return int(n), nil
	}
	for _, layout := range layouts {
		t, err := parseTime(layout, value)
		if err == nil {
			return int(t.Unix()), nil
		}
	}
	return 0, fmt.Errorf("invalid --%s %q (use %s)", name, value, Help)
}

func parseNow(value string, now time.Time) (int, bool, error) {
	lower := strings.ToLower(strings.TrimSpace(value))
	if !strings.HasPrefix(lower, "now") {
		return 0, false, nil
	}
	rest := strings.TrimSpace(lower[len("now"):])
	if rest == "" {
		return int(now.Unix()), true, nil
	}
	sign := rest[0]
	if sign != '+' && sign != '-' {
		return 0, true, fmt.Errorf("invalid relative time %q", value)
	}
	dur, err := parseDurationExpr(rest[1:])
	if err != nil {
		return 0, true, fmt.Errorf("invalid relative time %q: %w", value, err)
	}
	if sign == '-' {
		dur = -dur
	}
	return int(now.Add(dur).Unix()), true, nil
}

func parseDurationExpr(value string) (time.Duration, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	if value == "" {
		return 0, fmt.Errorf("missing duration")
	}
	var total time.Duration
	for value != "" {
		i := 0
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i == 0 {
			return 0, fmt.Errorf("missing number near %q", value)
		}
		n, err := strconv.ParseInt(value[:i], 10, 64)
		if err != nil {
			return 0, err
		}
		j := i
		for j < len(value) && ((value[j] >= 'a' && value[j] <= 'z') || (value[j] >= 'A' && value[j] <= 'Z')) {
			j++
		}
		if j == i {
			return 0, fmt.Errorf("missing unit near %q", value[i:])
		}
		unit := strings.ToLower(value[i:j])
		dur, ok := durationUnit(unit)
		if !ok {
			return 0, fmt.Errorf("unknown unit %q", unit)
		}
		total += time.Duration(n) * dur
		value = value[j:]
	}
	return total, nil
}

func durationUnit(unit string) (time.Duration, bool) {
	switch unit {
	case "ms":
		return time.Millisecond, true
	case "s":
		return time.Second, true
	case "m":
		return time.Minute, true
	case "h":
		return time.Hour, true
	case "d":
		return 24 * time.Hour, true
	case "w":
		return 7 * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func parseTime(layout, value string) (time.Time, error) {
	if layout == time.RFC3339 {
		return time.Parse(layout, value)
	}
	return time.ParseInLocation(layout, value, time.Local)
}
