package shared

import (
	"context"
	"fmt"

	"github.com/vika2603/100x-cli/api/futures"
)

// AttachOrderInput describes one attach-leg-to-order operation.
type AttachOrderInput struct {
	Symbol     string
	OrderID    string
	Leg        Leg
	Price      string
	PriceType  futures.StopTriggerType
	ClearOther bool
}

// BuildAttachOrderReq assembles a futures.StopOrderCloseReq, performing the
// read-modify-send compensation needed because the gateway endpoint takes
// both SL and TP fields in the same request.
//
// The opposite leg is preserved unless ClearOther is true. When the opposite
// leg is unset on the order (gateway reports "" or "-"), it is omitted rather
// than echoed back, since echoing the empty value trips code=20012/20014
// "stop loss/take profit price illegal".
func BuildAttachOrderReq(ctx context.Context, c *futures.Client, in AttachOrderInput) (futures.StopOrderCloseReq, error) {
	cur, err := c.Order.OrderDetail(ctx, futures.OrderDetailReq{Market: in.Symbol, OrderID: in.OrderID})
	if err != nil {
		return futures.StopOrderCloseReq{}, err
	}
	body := futures.StopOrderCloseReq{Market: in.Symbol, OrderID: in.OrderID}
	switch in.Leg {
	case LegSL:
		body.StopLossPrice = in.Price
		body.StopLossPriceType = in.PriceType
		if !in.ClearOther && priceSet(cur.TakeProfitPrice) {
			body.TakeProfitPrice = cur.TakeProfitPrice
			body.TakeProfitPriceType = futures.StopTriggerTypeLast
		}
	case LegTP:
		body.TakeProfitPrice = in.Price
		body.TakeProfitPriceType = in.PriceType
		if !in.ClearOther && priceSet(cur.StopLossPrice) {
			body.StopLossPrice = cur.StopLossPrice
			body.StopLossPriceType = futures.StopTriggerTypeLast
		}
	}
	return body, nil
}

// priceSet reports whether a price field returned by the gateway represents
// an actual value. The gateway sends "" or "-" for unset legs.
func priceSet(s string) bool {
	return s != "" && s != "-"
}

// AttachPositionInput describes one attach-leg-to-position operation.
type AttachPositionInput struct {
	Symbol     string
	PositionID string
	Leg        Leg
	Price      string
	PriceType  futures.StopTriggerType
	ClearOther bool
}

// BuildAttachPositionReq assembles a futures.StopClosePositionReq, performing
// the equivalent leg-preserve compensation for an open position.
func BuildAttachPositionReq(ctx context.Context, c *futures.Client, in AttachPositionInput) (futures.StopClosePositionReq, error) {
	pos, err := lookupPosition(ctx, c, in.Symbol, in.PositionID)
	if err != nil {
		return futures.StopClosePositionReq{}, err
	}
	body := futures.StopClosePositionReq{Market: in.Symbol, PositionID: in.PositionID}
	switch in.Leg {
	case LegSL:
		body.StopLossPrice = in.Price
		body.StopLossPriceType = in.PriceType
		if !in.ClearOther && pos != nil && priceSet(pos.TakeProfitPrice) {
			body.TakeProfitPrice = pos.TakeProfitPrice
			body.TakeProfitPriceType = pos.TakeProfitPriceType
		}
	case LegTP:
		body.TakeProfitPrice = in.Price
		body.TakeProfitPriceType = in.PriceType
		if !in.ClearOther && pos != nil && priceSet(pos.StopLossPrice) {
			body.StopLossPrice = pos.StopLossPrice
			body.StopLossPriceType = pos.StopLossPriceType
		}
	}
	return body, nil
}

func lookupPosition(ctx context.Context, c *futures.Client, market, positionID string) (*futures.PendingPositionDetail, error) {
	list, err := c.Position.PendingPosition(ctx, futures.PendingPositionReq{Market: market})
	if err != nil {
		return nil, err
	}
	for i := range list {
		if fmt.Sprint(list[i].PositionID) == positionID {
			return &list[i], nil
		}
	}
	return nil, nil
}
