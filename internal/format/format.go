// Package format maps futures-domain values to human-readable CLI cells.
//
// Enum values stay raw in JSON output; these helpers only shape human tables
// and object views.
package format

import (
	"strings"
	"time"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/output"
)

// Side colours BUY green and SELL red.
func Side(io *output.Renderer, s futures.Side) string {
	v := strings.ToUpper(s.String())
	switch s {
	case futures.SideBuy:
		return io.Positive(v)
	case futures.SideSell:
		return io.Negative(v)
	}
	return v
}

// OrderStatus colours by lifecycle:
//
//	FILLED                          → green
//	PENDING / PARTIAL               → yellow (work in flight)
//	CANCELED / PARTIAL-CANCELED     → muted gray
func OrderStatus(io *output.Renderer, s futures.OrderStatus) string {
	v := strings.ToUpper(s.String())
	switch s {
	case futures.OrderStatusFilled:
		return io.Positive(v)
	case futures.OrderStatusPending, futures.OrderStatusPartial:
		return io.Pending(v)
	case futures.OrderStatusCanceled, futures.OrderStatusPartialCanceled:
		return io.Subtle(v)
	}
	return v
}

// StopOrderStatus colours by lifecycle:
//
//	SUCCESS                         → green
//	UNTRIGGERED / UNACTIVATED       → muted gray (waiting)
//	CANCELED                        → muted gray
//	FAILED                          → red
func StopOrderStatus(io *output.Renderer, s futures.StopOrderStatus) string {
	v := strings.ToUpper(s.String())
	switch s {
	case futures.StopOrderStatusSuccess:
		return io.Positive(v)
	case futures.StopOrderStatusUnactivated, futures.StopOrderStatusUntriggered, futures.StopOrderStatusCanceled:
		return io.Subtle(v)
	case futures.StopOrderStatusFailed:
		return io.Negative(v)
	}
	return v
}

// PositionType colours both values cyan; CROSS and ISOLATED are not
// "good" or "bad", just informational categories worth highlighting.
func PositionType(io *output.Renderer, p futures.PositionType) string {
	return io.Accent(strings.ToUpper(p.String()))
}

// StopOrderType colours every variant cyan (informational).
func StopOrderType(io *output.Renderer, t futures.StopOrderType) string {
	return io.Accent(strings.ToUpper(t.String()))
}

// Enum uppercases string-valued gateway enums for human output.
func Enum(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return strings.ToUpper(value)
}

// UnixMillis formats gateway millisecond timestamps for human tables. JSON
// output keeps the original numeric value.
func UnixMillis(ms int) string {
	if ms <= 0 {
		return "-"
	}
	return time.UnixMilli(int64(ms)).Local().Format("2006-01-02 15:04:05")
}

// UnixAuto formats Unix seconds or milliseconds by inspecting magnitude.
func UnixAuto(ts int) string {
	if ts <= 0 {
		return "-"
	}
	if ts >= 1_000_000_000_000 {
		return time.UnixMilli(int64(ts)).Local().Format("2006-01-02 15:04:05")
	}
	return time.Unix(int64(ts), 0).Local().Format("2006-01-02 15:04:05")
}

// UnixSecondsFloat formats gateway second timestamps that arrive as float64.
func UnixSecondsFloat(sec float64) string {
	if sec <= 0 {
		return "-"
	}
	return time.Unix(int64(sec), 0).Local().Format("2006-01-02 15:04:05")
}
