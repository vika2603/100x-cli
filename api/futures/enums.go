package futures

// Side identifies the trade direction. Wire format: int.
type Side int

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

// TIF is the order time-in-force. Wire format: int.
type TIF int

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

// OrderStatus is the lifecycle state of a regular order. Wire format: int.
type OrderStatus int

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

// PositionType is cross vs isolated margining. Wire format: int.
type PositionType int

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
