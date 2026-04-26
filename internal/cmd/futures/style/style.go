// Package style maps futures-domain enum values to the Renderer's
// semantic colour primitives. Centralised here so every "side" cell
// gets the same green/red, every "filled" cell the same green, etc.
// — but the call sites still own which fields to show and how to
// label them.
package style

import (
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/output"
)

// Side colours BUY green and SELL red.
func Side(io *output.Renderer, s futures.Side) string {
	v := strings.ToUpper(s.String())
	switch s {
	case futures.SideBuy:
		return io.Success(v)
	case futures.SideSell:
		return io.Danger(v)
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
		return io.Success(v)
	case futures.OrderStatusPending, futures.OrderStatusPartial:
		return io.Warning(v)
	case futures.OrderStatusCanceled, futures.OrderStatusPartialCanceled:
		return io.Muted(v)
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
		return io.Success(v)
	case futures.StopOrderStatusUnactivated, futures.StopOrderStatusUntriggered, futures.StopOrderStatusCanceled:
		return io.Muted(v)
	case futures.StopOrderStatusFailed:
		return io.Danger(v)
	}
	return v
}

// PositionType colours both values cyan; CROSS and ISOLATED are not
// "good" or "bad", just informational categories worth highlighting.
func PositionType(io *output.Renderer, p futures.PositionType) string {
	return io.Info(strings.ToUpper(p.String()))
}

// StopOrderType colours every variant cyan (informational).
func StopOrderType(io *output.Renderer, t futures.StopOrderType) string {
	return io.Info(strings.ToUpper(t.String()))
}
