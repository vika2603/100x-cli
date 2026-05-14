package futures

import (
	"fmt"
	"strings"
)

// Side identifies the trade direction. Wire format: int.
type Side int

// Side values.
const (
	SideSell Side = 1
	SideBuy  Side = 2
)

func (s Side) String() string {
	switch s {
	case SideSell:
		return "sell"
	case SideBuy:
		return "buy"
	default:
		return "unknown"
	}
}

// ParseSide maps a case-insensitive textual side ("buy"/"b" or "sell"/"s")
// to a Side value. Empty input is rejected.
func ParseSide(s string) (Side, error) {
	switch strings.ToUpper(s) {
	case "BUY", "B":
		return SideBuy, nil
	case "SELL", "S":
		return SideSell, nil
	}
	return 0, fmt.Errorf("unknown side %q (want buy|sell)", s)
}

// TIF is the order time-in-force. Wire format: int.
type TIF int

// TIF values.
const (
	TIFGTC      TIF = 0
	TIFFOK      TIF = 1
	TIFIOC      TIF = 2
	TIFPostOnly TIF = 3
)

func (t TIF) String() string {
	switch t {
	case TIFGTC:
		return "GTC"
	case TIFFOK:
		return "FOK"
	case TIFIOC:
		return "IOC"
	case TIFPostOnly:
		return "PostOnly"
	default:
		return "unknown"
	}
}

// ParseTIF maps a case-insensitive textual time-in-force to a TIF value.
// Empty input defaults to GTC. POST_ONLY accepts "POST_ONLY", "POSTONLY",
// or "PO".
func ParseTIF(s string) (TIF, error) {
	switch strings.ToUpper(s) {
	case "", "GTC":
		return TIFGTC, nil
	case "FOK":
		return TIFFOK, nil
	case "IOC":
		return TIFIOC, nil
	case "POST_ONLY", "POSTONLY", "PO":
		return TIFPostOnly, nil
	}
	return 0, fmt.Errorf("unknown --tif %q (want GTC|FOK|IOC|POST_ONLY)", s)
}

// OrderStatus is the lifecycle state of a regular order. Wire format: int.
type OrderStatus int

// OrderStatus values.
const (
	OrderStatusPending         OrderStatus = 1
	OrderStatusPartial         OrderStatus = 2
	OrderStatusFilled          OrderStatus = 3
	OrderStatusPartialCanceled OrderStatus = 4
	OrderStatusCanceled        OrderStatus = 5
)

func (s OrderStatus) String() string {
	switch s {
	case OrderStatusPending:
		return "pending"
	case OrderStatusPartial:
		return "partial"
	case OrderStatusFilled:
		return "filled"
	case OrderStatusPartialCanceled:
		return "partial-canceled"
	case OrderStatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// StopOrderStatus is the lifecycle state of a condition order. Wire format: int.
type StopOrderStatus int

// StopOrderStatus values.
const (
	StopOrderStatusUnactivated StopOrderStatus = 1
	StopOrderStatusUntriggered StopOrderStatus = 2
	StopOrderStatusSuccess     StopOrderStatus = 3
	StopOrderStatusCanceled    StopOrderStatus = 4
	StopOrderStatusFailed      StopOrderStatus = 5
)

func (s StopOrderStatus) String() string {
	switch s {
	case StopOrderStatusUnactivated:
		return "unactivated"
	case StopOrderStatusUntriggered:
		return "untriggered"
	case StopOrderStatusSuccess:
		return "success"
	case StopOrderStatusCanceled:
		return "canceled"
	case StopOrderStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// StopOrderType (wire field: contract_order_type) categorises a condition order.
type StopOrderType int

// StopOrderType values.
const (
	StopOrderTypeStandalone         StopOrderType = 0
	StopOrderTypePositionTakeProfit StopOrderType = 1
	StopOrderTypePositionStopLoss   StopOrderType = 2
	StopOrderTypeOrderTakeProfit    StopOrderType = 3
	StopOrderTypeOrderStopLoss      StopOrderType = 4
)

func (t StopOrderType) String() string {
	switch t {
	case StopOrderTypeStandalone:
		return "standalone"
	case StopOrderTypePositionTakeProfit:
		return "position-tp"
	case StopOrderTypePositionStopLoss:
		return "position-sl"
	case StopOrderTypeOrderTakeProfit:
		return "order-tp"
	case StopOrderTypeOrderStopLoss:
		return "order-sl"
	default:
		return "unknown"
	}
}

// StopTriggerType selects which price feed a trigger watches. Wire format: int.
type StopTriggerType int

// StopTriggerType values.
const (
	StopTriggerTypeLast  StopTriggerType = 1
	StopTriggerTypeIndex StopTriggerType = 2
	StopTriggerTypeMark  StopTriggerType = 3
)

func (t StopTriggerType) String() string {
	switch t {
	case StopTriggerTypeLast:
		return "last"
	case StopTriggerTypeIndex:
		return "index"
	case StopTriggerTypeMark:
		return "mark"
	default:
		return "unknown"
	}
}

// ParseStopTriggerType maps a case-insensitive feed name ("last"/"index"/
// "mark") to a StopTriggerType. Empty input defaults to Last.
func ParseStopTriggerType(s string) (StopTriggerType, error) {
	switch strings.ToUpper(s) {
	case "", "LAST":
		return StopTriggerTypeLast, nil
	case "INDEX":
		return StopTriggerTypeIndex, nil
	case "MARK":
		return StopTriggerTypeMark, nil
	}
	return 0, fmt.Errorf("unknown trigger price type %q (want LAST|INDEX|MARK)", s)
}

// PositionType is cross vs isolated margining. Wire format: int.
type PositionType int

// PositionType values.
const (
	PositionTypeCross    PositionType = 1
	PositionTypeIsolated PositionType = 2
)

func (p PositionType) String() string {
	switch p {
	case PositionTypeCross:
		return "cross"
	case PositionTypeIsolated:
		return "isolated"
	default:
		return "unknown"
	}
}

// MarginAction selects whether an AdjustPositionMarginReq adds or removes margin.
type MarginAction int

// MarginAction values.
const (
	MarginActionAdd    MarginAction = 1
	MarginActionRemove MarginAction = 2
)

func (a MarginAction) String() string {
	switch a {
	case MarginActionAdd:
		return "add"
	case MarginActionRemove:
		return "remove"
	default:
		return "unknown"
	}
}
