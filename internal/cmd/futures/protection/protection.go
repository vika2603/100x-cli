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
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/clierr"
)

// Kind names which side of the protection a request is updating.
type Kind int

const (
	// KindStopLoss is the stop-loss side.
	KindStopLoss Kind = iota
	// KindTakeProfit is the take-profit side.
	KindTakeProfit
)

// String returns the canonical CLI label.
func (k Kind) String() string {
	if k == KindTakeProfit {
		return "TP"
	}
	return "SL"
}

// ParseKind accepts the user-typed CLI labels.
func ParseKind(s string) (Kind, error) {
	switch strings.ToUpper(s) {
	case "SL", "STOP-LOSS":
		return KindStopLoss, nil
	case "TP", "TAKE-PROFIT":
		return KindTakeProfit, nil
	}
	return 0, clierr.Usagef("unknown protection kind %q (want SL|TP)", s)
}

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
// hides three branches: editing an existing standalone trigger via
// EditStopOrder, the documented two-call TP-while-preserving-SL quirk, and
// full-body StopOrderClose (used both for trigger attach with side
// preservation and for order edit's re-attach of both sides).
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

	// Adding TP while keeping an existing standalone SL is the documented
	// two-call quirk: the first call sets TP alone, the second restates SL
	// alongside the new TP.
	if tpChanged && !slChanged && want.TP.Set() && current.TP.TriggerID == "" && current.SL.TriggerID != "" {
		if _, err := c.Order.StopOrderClose(ctx, futures.StopOrderCloseReq{
			Market: o.Symbol, OrderID: o.OrderID,
			TakeProfitPrice: want.TP.Price, TakeProfitPriceType: want.TP.PriceType,
		}); err != nil {
			return err
		}
		_, err := c.Order.StopOrderClose(ctx, futures.StopOrderCloseReq{
			Market: o.Symbol, OrderID: o.OrderID,
			StopLossPrice: current.SL.Price, StopLossPriceType: current.SL.PriceType,
			TakeProfitPrice: want.TP.Price, TakeProfitPriceType: want.TP.PriceType,
		})
		return err
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
	if want.SL.Set() && detail.StopLossPrice != want.SL.Price {
		return fmt.Errorf("gateway accepted attach but SL on order %s is %q, want %q", o.OrderID, detail.StopLossPrice, want.SL.Price)
	}
	if want.TP.Set() && detail.TakeProfitPrice != want.TP.Price {
		return fmt.Errorf("gateway accepted attach but TP on order %s is %q, want %q", o.OrderID, detail.TakeProfitPrice, want.TP.Price)
	}
	return nil
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
// position. There is no two-call quirk here; the gateway accepts both sides
// in one StopClosePosition call.
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
	if want.SL.Set() && pos.StopLossPrice != want.SL.Price {
		return fmt.Errorf("gateway accepted attach but SL on position %s is %q, want %q", p.PositionID, pos.StopLossPrice, want.SL.Price)
	}
	if want.TP.Set() && pos.TakeProfitPrice != want.TP.Price {
		return fmt.Errorf("gateway accepted attach but TP on position %s is %q, want %q", p.PositionID, pos.TakeProfitPrice, want.TP.Price)
	}
	return nil
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
