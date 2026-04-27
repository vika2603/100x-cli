package futures

import "context"

// PositionClient covers every endpoint under /open/api/v2/position.
type PositionClient struct {
	doer Doer
}

// PendingPositionReq filters the open-positions list.
type PendingPositionReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// PendingPosition lists currently open positions.
func (c *PositionClient) PendingPosition(ctx context.Context, req PendingPositionReq) ([]PendingPositionDetail, error) {
	var resp []PendingPositionDetail
	if err := c.doer.Get(ctx, "/open/api/v2/position/pending", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AdjustPositionMarginReq moves margin in or out of a position via POST /position/margin.
type AdjustPositionMarginReq struct {
	Market   string       `url:"market,omitempty" json:"market,omitempty"`
	Type     MarginAction `url:"type,omitempty" json:"type,omitempty"`
	Quantity string       `url:"quantity,omitempty" json:"quantity,omitempty"`
}

// AdjustPositionMargin adds or removes margin.
func (c *PositionClient) AdjustPositionMargin(ctx context.Context, req AdjustPositionMarginReq) (*PendingPositionDetail, error) {
	var resp PendingPositionDetail
	if err := c.doer.Post(ctx, "/open/api/v2/position/margin", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PositionAdjustableMarginReq queries the margin currently movable.
type PositionAdjustableMarginReq struct {
	PositionID int    `url:"position_id,omitempty" json:"position_id,omitempty"`
	Market     string `url:"market,omitempty" json:"market,omitempty"`
}

// PositionAdjustableMargin returns the margin available to add or remove.
func (c *PositionClient) PositionAdjustableMargin(ctx context.Context, req PositionAdjustableMarginReq) (*PositionAdjustableMarginResp, error) {
	var resp PositionAdjustableMarginResp
	if err := c.doer.Get(ctx, "/open/api/v2/position/margin", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitClosePositionReq closes part of a position via a limit order.
type LimitClosePositionReq struct {
	Market     string `url:"market,omitempty" json:"market,omitempty"`
	PositionID string `url:"position_id,omitempty" json:"position_id,omitempty"`
	Quantity   string `url:"quantity,omitempty" json:"quantity,omitempty"`
	Price      string `url:"price,omitempty" json:"price,omitempty"`
	ClientOID  string `url:"client_oid,omitempty" json:"client_oid,omitempty"`
}

// LimitClosePositionResp wraps the OrderItem returned by /position/close/limit.
type LimitClosePositionResp struct{ OrderItem }

// LimitClosePosition submits a limit close.
func (c *PositionClient) LimitClosePosition(ctx context.Context, req LimitClosePositionReq) (*LimitClosePositionResp, error) {
	var resp LimitClosePositionResp
	if err := c.doer.Post(ctx, "/open/api/v2/position/close/limit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketClosePositionReq closes part of a position via a market order.
type MarketClosePositionReq struct {
	Market     string `url:"market,omitempty" json:"market,omitempty"`
	PositionID string `url:"position_id,omitempty" json:"position_id,omitempty"`
	Quantity   string `url:"quantity,omitempty" json:"quantity,omitempty"`
	ClientOID  string `url:"client_oid,omitempty" json:"client_oid,omitempty"`
}

// MarketClosePositionResp wraps the OrderItem returned by /position/close/market.
type MarketClosePositionResp struct{ OrderItem }

// MarketClosePosition submits a market close.
func (c *PositionClient) MarketClosePosition(ctx context.Context, req MarketClosePositionReq) (*MarketClosePositionResp, error) {
	var resp MarketClosePositionResp
	if err := c.doer.Post(ctx, "/open/api/v2/position/close/market", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopClosePositionReq attaches SL/TP to a position via POST /position/close/stop.
//
// The endpoint requires both SL and TP fields to be sent together; callers
// preserving one side while updating the other must read current values and
// pass both fields.
type StopClosePositionReq struct {
	Market              string          `url:"market,omitempty" json:"market,omitempty"`
	PositionID          string          `url:"position_id,omitempty" json:"position_id,omitempty"`
	StopLossPrice       string          `url:"stop_loss_price,omitempty" json:"stop_loss_price,omitempty"`
	StopLossPriceType   StopTriggerType `url:"stop_loss_price_type,omitempty" json:"stop_loss_price_type,omitempty"`
	TakeProfitPrice     string          `url:"take_profit_price,omitempty" json:"take_profit_price,omitempty"`
	TakeProfitPriceType StopTriggerType `url:"take_profit_price_type,omitempty" json:"take_profit_price_type,omitempty"`
}

// StopClosePositionResp is the empty response of /position/close/stop.
type StopClosePositionResp struct{}

// StopClosePosition attaches SL/TP to an open position.
func (c *PositionClient) StopClosePosition(ctx context.Context, req StopClosePositionReq) (*StopClosePositionResp, error) {
	var resp StopClosePositionResp
	if err := c.doer.Post(ctx, "/open/api/v2/position/close/stop", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitAddPositionReq tops up a position via a limit order.
type LimitAddPositionReq struct {
	Market     string `url:"market,omitempty" json:"market,omitempty"`
	PositionID string `url:"position_id,omitempty" json:"position_id,omitempty"`
	Quantity   string `url:"quantity,omitempty" json:"quantity,omitempty"`
	Price      string `url:"price,omitempty" json:"price,omitempty"`
	ClientOID  string `url:"client_oid,omitempty" json:"client_oid,omitempty"`
}

// LimitAddPositionResp wraps the OrderItem returned by /position/add/limit.
type LimitAddPositionResp struct{ OrderItem }

// LimitAddPosition submits a limit add.
func (c *PositionClient) LimitAddPosition(ctx context.Context, req LimitAddPositionReq) (*LimitAddPositionResp, error) {
	var resp LimitAddPositionResp
	if err := c.doer.Post(ctx, "/open/api/v2/position/add/limit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketAddPositionReq tops up a position via a market order.
type MarketAddPositionReq struct {
	Market     string `url:"market,omitempty" json:"market,omitempty"`
	PositionID string `url:"position_id,omitempty" json:"position_id,omitempty"`
	Quantity   string `url:"quantity,omitempty" json:"quantity,omitempty"`
	ClientOID  string `url:"client_oid,omitempty" json:"client_oid,omitempty"`
}

// MarketAddPositionResp wraps the OrderItem returned by /position/add/market.
type MarketAddPositionResp struct{ OrderItem }

// MarketAddPosition submits a market add.
func (c *PositionClient) MarketAddPosition(ctx context.Context, req MarketAddPositionReq) (*MarketAddPositionResp, error) {
	var resp MarketAddPositionResp
	if err := c.doer.Post(ctx, "/open/api/v2/position/add/market", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PositionHistoryReq filters the closed-positions history.
type PositionHistoryReq struct {
	Market    string `url:"market,omitempty" json:"market,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
	Page      int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize  int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// PositionHistoryResp is the paginated closed-positions response.
type PositionHistoryResp struct {
	Records  []FinishedPositionDetail `url:"records" json:"records"`
	PageSize int                      `url:"page_size" json:"page_size"`
	Page     int                      `url:"page" json:"page"`
}

// PositionHistory lists past positions.
func (c *PositionClient) PositionHistory(ctx context.Context, req PositionHistoryReq) (*PositionHistoryResp, error) {
	var resp PositionHistoryResp
	if err := c.doer.Get(ctx, "/open/api/v2/position/history", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
