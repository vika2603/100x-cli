// Package protection models the SL/TP stop-protection layer that hangs off
// a pending order or an open position.
//
// Both `trigger attach` and `order edit` mutate that layer. They share a
// non-trivial gateway dance: read the current SL and TP (some on the parent
// order/position record, some as standalone StopOrder records), preserve
// whichever side was not requested, and verify the result. This package owns
// that dance so verb files stay short.
//
// The interface is intentionally a state-transition function:
//
//	current, _ := target.Inspect(ctx, c)
//	want := mutate(current)
//	target.Apply(ctx, c, current, want)
//	target.Verify(ctx, c, want)
//
// `trigger attach` builds want from current with one of SL/TP modified;
// `order edit` builds want from a freshly rebooked order's empty State and
// the old protection it wants to re-attach.
package protection

import (
	"context"
	"fmt"
	"strconv"

	"github.com/vika2603/100x-cli/api/futures"
)

// Stop is one observed SL or TP setting.
//
// Price is "" or "-" when the side is unset (the gateway uses both forms).
// PriceType is the trigger feed for that side. TriggerID is the
// contract_order_id of the standalone StopOrder pegging the side, when one
// exists; when the side lives only on the parent order/position record,
// TriggerID is empty.
type Stop struct {
	Price     string
	PriceType futures.StopTriggerType
	TriggerID string
}

// Set reports whether the side carries a real price.
func (s Stop) Set() bool {
	return priceSet(s.Price)
}

// State is the SL/TP protection observed on an order or position.
type State struct {
	SL Stop
	TP Stop
	// CrossOrderConflict is true when another pending order on the same
	// position has its own active SL/TP triggers. The gateway scopes
	// order-level SL/TP at the position, so attaching here would silently
	// overlap with the other order's triggers. Always false for
	// Position.Inspect.
	CrossOrderConflict bool
}

// HasAny reports whether any protection is currently observed.
func (s State) HasAny() bool {
	return s.SL.Set() || s.TP.Set()
}

// Order is the protection layer for one pending limit order.
type Order struct {
	Symbol  string
	OrderID string
}

// Inspect reads the current protection state for the order, including any
// standalone StopOrder triggers tied to it and a CrossOrderConflict flag for
// sibling orders sharing the position.
func (o Order) Inspect(ctx context.Context, c *futures.Client) (State, error) {
	detail, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: o.Symbol, OrderID: o.OrderID})
	if err != nil {
		return State{}, err
	}
	s := State{}
	if priceSet(detail.StopLossPrice) {
		s.SL = Stop{Price: detail.StopLossPrice, PriceType: futures.StopTriggerTypeLast}
	}
	if priceSet(detail.TakeProfitPrice) {
		s.TP = Stop{Price: detail.TakeProfitPrice, PriceType: futures.StopTriggerTypeLast}
	}
	stops, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: o.Symbol, Page: 1, PageSize: 20})
	if err != nil {
		return s, err
	}
	for i := range stops.Records {
		st := stops.Records[i]
		if st.PositionID == detail.PositionID && st.OrderID != detail.OrderID {
			s.CrossOrderConflict = true
		}
		if st.OrderID != detail.OrderID {
			continue
		}
		switch st.ContractOrderType {
		case futures.StopOrderTypeOrderStopLoss:
			s.SL = Stop{Price: st.TriggerPrice, PriceType: st.TriggerType, TriggerID: st.ContractOrderID}
		case futures.StopOrderTypeOrderTakeProfit:
			s.TP = Stop{Price: st.TriggerPrice, PriceType: st.TriggerType, TriggerID: st.ContractOrderID}
		}
	}
	return s, nil
}

// Apply emits the minimum gateway sequence to make want the new state. It
// routes through five branches:
//
//   - single-side change with an existing standalone TriggerID → /order/stop/edit
//   - both-side change with both standalone TriggerIDs present → 2× /order/stop/edit
//   - the documented two-call TP-while-preserving-SL quirk
//   - mixed: one side already standalone, the other side fresh — gateway's
//     /order/close/stop SL block early-returns once it has run an update,
//     so it never reaches the TP block. We surface an explicit error
//     instead of letting the request silently lose protection.
//   - cold-start full-body StopOrderClose for everything else (first-time
//     attach, ClearOther, order edit's re-attach of both sides).
func (o Order) Apply(ctx context.Context, c *futures.Client, current, want State) error {
	slChanged := want.SL != current.SL
	tpChanged := want.TP != current.TP
	if !slChanged && !tpChanged {
		return nil
	}

	// Edit a standalone trigger in place when only one side moves and that
	// side is already pegged server-side.
	if slChanged && !tpChanged && current.SL.TriggerID != "" && want.SL.Set() {
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: o.Symbol, StopOrderID: current.SL.TriggerID,
			StopPrice: want.SL.Price, StopPriceType: want.SL.PriceType,
		})
		return err
	}
	if tpChanged && !slChanged && current.TP.TriggerID != "" && want.TP.Set() {
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: o.Symbol, StopOrderID: current.TP.TriggerID,
			StopPrice: want.TP.Price, StopPriceType: want.TP.PriceType,
		})
		return err
	}

	// Both sides change and both already exist as standalone triggers: edit
	// each via its own trigger id. /order/close/stop's update path early-
	// returns after handling SL, so a single full-body call would never
	// touch TP.
	if slChanged && tpChanged && current.SL.TriggerID != "" && current.TP.TriggerID != "" && want.SL.Set() && want.TP.Set() {
		if _, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: o.Symbol, StopOrderID: current.SL.TriggerID,
			StopPrice: want.SL.Price, StopPriceType: want.SL.PriceType,
		}); err != nil {
			return err
		}
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: o.Symbol, StopOrderID: current.TP.TriggerID,
			StopPrice: want.TP.Price, StopPriceType: want.TP.PriceType,
		})
		return err
	}

	// SL already standalone + want to also set TP: the gateway's
	// /order/close/stop handler runs the SL block first; finding a
	// standalone SL it goes through ConditionOrderUpdate and early-returns
	// before the TP block can fire, so the requested TP is silently
	// dropped. The reverse (TP standalone + want SL too) works because
	// the SL block then takes the fresh-entrust path with no early
	// return, and the TP block subsequently updates. Surface the broken
	// direction as an explicit error pointing at the manual recovery.
	if want.SL.Set() && want.TP.Set() && current.SL.TriggerID != "" && current.TP.TriggerID == "" {
		return fmt.Errorf("cannot set SL and TP on order %s in one call: SL is already a standalone trigger; cancel it first via `100x futures trigger cancel %s %s` and retry", o.OrderID, o.Symbol, current.SL.TriggerID)
	}

	// Cold-start full-body StopOrderClose. Used for first-time attach with
	// side-preservation, ClearOther, and order edit's re-attach of both
	// sides on a freshly rebooked order.
	body := futures.StopOrderCloseReq{Market: o.Symbol, OrderID: o.OrderID}
	if want.SL.Set() {
		body.StopLossPrice = want.SL.Price
		body.StopLossPriceType = want.SL.PriceType
	}
	if want.TP.Set() {
		body.TakeProfitPrice = want.TP.Price
		body.TakeProfitPriceType = want.TP.PriceType
	}
	_, err := c.Order.StopOrderClose(ctx, body)
	return err
}

// Verify reads the order back and confirms each Set side of want matches.
// Unset sides are not checked.
func (o Order) Verify(ctx context.Context, c *futures.Client, want State) error {
	detail, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: o.Symbol, OrderID: o.OrderID})
	if err != nil {
		return err
	}
	if err := verifySide("order", o.OrderID, "SL", detail.StopLossPrice, want.SL); err != nil {
		return err
	}
	return verifySide("order", o.OrderID, "TP", detail.TakeProfitPrice, want.TP)
}

// Position is the protection layer for one open position.
type Position struct {
	Symbol     string
	PositionID string
}

// Inspect reads the current protection state for the position, including any
// standalone StopOrder triggers tied to it.
func (p Position) Inspect(ctx context.Context, c *futures.Client) (State, error) {
	pos, err := lookupPosition(ctx, c, p.Symbol, p.PositionID)
	if err != nil {
		return State{}, err
	}
	s := State{}
	if pos != nil {
		if priceSet(pos.StopLossPrice) {
			s.SL = Stop{Price: pos.StopLossPrice, PriceType: pos.StopLossPriceType}
		}
		if priceSet(pos.TakeProfitPrice) {
			s.TP = Stop{Price: pos.TakeProfitPrice, PriceType: pos.TakeProfitPriceType}
		}
	}
	posID, _ := strconv.ParseInt(p.PositionID, 10, 64)
	stops, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: p.Symbol, Page: 1, PageSize: 20})
	if err != nil {
		return s, err
	}
	for i := range stops.Records {
		st := stops.Records[i]
		if st.PositionID != posID {
			continue
		}
		switch st.ContractOrderType {
		case futures.StopOrderTypePositionStopLoss:
			s.SL = Stop{Price: st.TriggerPrice, PriceType: st.TriggerType, TriggerID: st.ContractOrderID}
		case futures.StopOrderTypePositionTakeProfit:
			s.TP = Stop{Price: st.TriggerPrice, PriceType: st.TriggerType, TriggerID: st.ContractOrderID}
		}
	}
	return s, nil
}

// Apply emits the minimum gateway sequence to make want the new state on the
// position. Same routing as Order.Apply except there is no TP-preserving-SL
// two-call quirk on the position endpoint.
func (p Position) Apply(ctx context.Context, c *futures.Client, current, want State) error {
	slChanged := want.SL != current.SL
	tpChanged := want.TP != current.TP
	if !slChanged && !tpChanged {
		return nil
	}

	if slChanged && !tpChanged && current.SL.TriggerID != "" && want.SL.Set() {
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: p.Symbol, StopOrderID: current.SL.TriggerID,
			StopPrice: want.SL.Price, StopPriceType: want.SL.PriceType,
		})
		return err
	}
	if tpChanged && !slChanged && current.TP.TriggerID != "" && want.TP.Set() {
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: p.Symbol, StopOrderID: current.TP.TriggerID,
			StopPrice: want.TP.Price, StopPriceType: want.TP.PriceType,
		})
		return err
	}

	if slChanged && tpChanged && current.SL.TriggerID != "" && current.TP.TriggerID != "" && want.SL.Set() && want.TP.Set() {
		if _, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: p.Symbol, StopOrderID: current.SL.TriggerID,
			StopPrice: want.SL.Price, StopPriceType: want.SL.PriceType,
		}); err != nil {
			return err
		}
		_, err := c.Order.EditStopOrder(ctx, futures.StopOrderEditReq{
			Market: p.Symbol, StopOrderID: current.TP.TriggerID,
			StopPrice: want.TP.Price, StopPriceType: want.TP.PriceType,
		})
		return err
	}

	// /position/close/stop runs its SL and TP blocks in separate
	// goroutines, so unlike /order/close/stop there is no early-return bug
	// when one side is already standalone — both blocks always fire and
	// take their respective update / entrust path. Cold-start handles
	// every remaining shape; Verify catches per-side silent failures.
	body := futures.StopClosePositionReq{Market: p.Symbol, PositionID: p.PositionID}
	if want.SL.Set() {
		body.StopLossPrice = want.SL.Price
		body.StopLossPriceType = want.SL.PriceType
	}
	if want.TP.Set() {
		body.TakeProfitPrice = want.TP.Price
		body.TakeProfitPriceType = want.TP.PriceType
	}
	_, err := c.Position.StopClosePosition(ctx, body)
	return err
}

// Verify reads the position back and confirms each Set side of want matches.
func (p Position) Verify(ctx context.Context, c *futures.Client, want State) error {
	pos, err := lookupPosition(ctx, c, p.Symbol, p.PositionID)
	if err != nil {
		return err
	}
	if pos == nil {
		return fmt.Errorf("position %s not found after attach", p.PositionID)
	}
	if err := verifySide("position", p.PositionID, "SL", pos.StopLossPrice, want.SL); err != nil {
		return err
	}
	return verifySide("position", p.PositionID, "TP", pos.TakeProfitPrice, want.TP)
}

// verifySide compares the gateway's read-back value for one side against
// what the caller wanted. The gateway returns "-" or "" when a side is
// unset, so a mismatch where got is unset means the gateway accepted the
// request but did not persist that side — most often because the price
// fell on the wrong side of the order/position direction or because the
// per-user condition-order limit is full. Surface that explicitly instead
// of letting the user decode a literal `is "-"` string.
func verifySide(scope, id, side, got string, want Stop) error {
	if !want.Set() {
		return nil
	}
	if got == want.Price {
		return nil
	}
	if !priceSet(got) {
		return fmt.Errorf("gateway accepted attach but %s on %s %s was not applied (requested %s); check the price is on the profit/loss side of the %s direction, or that the per-user condition-order limit is not full", side, scope, id, want.Price, scope)
	}
	return fmt.Errorf("gateway accepted attach but %s on %s %s is %q, want %q", side, scope, id, got, want.Price)
}

// IsAttached reports whether triggerID is an attached SL/TP trigger. The
// gateway's EditStopOrder endpoint rejects standalone triggers with
// code=10066; callers should use IsAttached up-front to convert that into a
// friendly error. A trigger that is no longer pending returns (false, nil).
func IsAttached(ctx context.Context, c *futures.Client, market, triggerID string) (bool, error) {
	stops, err := c.Order.PendingStopOrder(ctx, futures.PendingStopOrderReq{Market: market, Page: 1, PageSize: 20})
	if err != nil {
		return false, err
	}
	for _, st := range stops.Records {
		if st.ContractOrderID == triggerID {
			return st.ContractOrderType != futures.StopOrderTypeStandalone, nil
		}
	}
	return false, nil
}

func priceSet(s string) bool {
	return s != "" && s != "-"
}

func lookupPosition(ctx context.Context, c *futures.Client, market, positionID string) (*futures.PendingPositionDetail, error) {
	list, err := c.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: market})
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strconv.Itoa(list[i].PositionID) == positionID {
			return &list[i], nil
		}
	}
	return nil, nil
}
