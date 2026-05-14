package futures

import "context"

// OrderClient covers every endpoint under /open/api/v2/order.
type OrderClient struct {
	doer Doer
}

// LimitOrderReq submits a limit order via POST /order/limit.
type LimitOrderReq struct {
	Market              string          `url:"market,omitempty" json:"market,omitempty"`
	Side                Side            `url:"side,omitempty" json:"side,omitempty"`
	Price               string          `url:"price,omitempty" json:"price,omitempty"`
	Quantity            string          `url:"quantity,omitempty" json:"quantity,omitempty"`
	ClientOID           string          `url:"client_oid,omitempty" json:"client_oid,omitempty"`
	IsStop              bool            `url:"is_stop,omitempty" json:"is_stop,omitempty"`
	TIF                 TIF             `url:"tif,omitempty" json:"tif,omitempty"`
	StopLossPrice       string          `url:"stop_loss_price,omitempty" json:"stop_loss_price,omitempty"`
	StopLossPriceType   StopTriggerType `url:"stop_loss_price_type,omitempty" json:"stop_loss_price_type,omitempty"`
	TakeProfitPrice     string          `url:"take_profit_price,omitempty" json:"take_profit_price,omitempty"`
	TakeProfitPriceType StopTriggerType `url:"take_profit_price_type,omitempty" json:"take_profit_price_type,omitempty"`
}

// LimitOrderResp wraps the OrderItem returned by /order/limit.
type LimitOrderResp struct{ OrderItem }

// LimitOrder submits a limit order.
func (c *OrderClient) LimitOrder(ctx context.Context, req LimitOrderReq) (*LimitOrderResp, error) {
	var resp LimitOrderResp
	if err := c.doer.Post(ctx, apiBase+"/order/limit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketOrderReq submits a market order via POST /order/market.
type MarketOrderReq struct {
	Market              string          `url:"market,omitempty" json:"market,omitempty"`
	Side                Side            `url:"side,omitempty" json:"side,omitempty"`
	Quantity            string          `url:"quantity,omitempty" json:"quantity,omitempty"`
	ClientOID           string          `url:"client_oid,omitempty" json:"client_oid,omitempty"`
	IsStop              bool            `url:"is_stop,omitempty" json:"is_stop,omitempty"`
	StopLossPrice       string          `url:"stop_loss_price,omitempty" json:"stop_loss_price,omitempty"`
	StopLossPriceType   StopTriggerType `url:"stop_loss_price_type,omitempty" json:"stop_loss_price_type,omitempty"`
	TakeProfitPrice     string          `url:"take_profit_price,omitempty" json:"take_profit_price,omitempty"`
	TakeProfitPriceType StopTriggerType `url:"take_profit_price_type,omitempty" json:"take_profit_price_type,omitempty"`
}

// MarketOrderResp wraps the OrderItem returned by /order/market.
type MarketOrderResp struct{ OrderItem }

// MarketOrder submits a market order.
func (c *OrderClient) MarketOrder(ctx context.Context, req MarketOrderReq) (*MarketOrderResp, error) {
	var resp MarketOrderResp
	if err := c.doer.Post(ctx, apiBase+"/order/market", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopOrderReq submits a standalone condition order via POST /order/stop.
//
// Despite the name, `CutPrice` is the current-price snapshot that the
// gateway validates the trigger against, not a post-trigger cutoff. It must
// be the live ticker value matching `StopPriceType` (Last / IndexPrice /
// SignPrice); omitting it or sending a value that disagrees with the live
// feed rejects with code=20021 "current price illegal". The CLI fills it
// from /market/state when the user does not pass --current-price.
type StopOrderReq struct {
	Market        string          `url:"market,omitempty" json:"market,omitempty"`
	Side          Side            `url:"side,omitempty" json:"side,omitempty"`
	OrderPrice    string          `url:"order_price,omitempty" json:"order_price,omitempty"`
	StopPrice     string          `url:"stop_price,omitempty" json:"stop_price,omitempty"`
	CutPrice      string          `url:"cut_price,omitempty" json:"cut_price,omitempty"`
	StopPriceType StopTriggerType `url:"stop_price_type,omitempty" json:"stop_price_type,omitempty"`
	Quantity      string          `url:"quantity,omitempty" json:"quantity,omitempty"`
}

// StopOrderResp is the empty response of /order/stop.
type StopOrderResp struct{}

// StopOrder submits a standalone condition order.
func (c *OrderClient) StopOrder(ctx context.Context, req StopOrderReq) (*StopOrderResp, error) {
	var resp StopOrderResp
	if err := c.doer.Post(ctx, apiBase+"/order/stop", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitOrderCancelReq cancels one regular order via POST /order/cancel.
type LimitOrderCancelReq struct {
	Market  string `url:"market,omitempty" json:"market,omitempty"`
	OrderID string `url:"order_id,omitempty" json:"order_id,omitempty"`
}

// LimitOrderCancelResp wraps the OrderItem returned by /order/cancel.
type LimitOrderCancelResp struct{ OrderItem }

// CancelOrder cancels one regular (limit/market) order.
func (c *OrderClient) CancelOrder(ctx context.Context, req LimitOrderCancelReq) (*LimitOrderCancelResp, error) {
	var resp LimitOrderCancelResp
	if err := c.doer.Post(ctx, apiBase+"/order/cancel", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitOrderCancelAllReq cancels every regular order in one market.
type LimitOrderCancelAllReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// LimitOrderCancelAllResp is the empty response of /order/cancel/all.
type LimitOrderCancelAllResp struct{}

// CancelAllOrder cancels every open regular order in one market.
func (c *OrderClient) CancelAllOrder(ctx context.Context, req LimitOrderCancelAllReq) (*LimitOrderCancelAllResp, error) {
	var resp LimitOrderCancelAllResp
	if err := c.doer.Post(ctx, apiBase+"/order/cancel/all", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitOrderCancelBatchReq cancels multiple orders by id via POST /order/cancel/batch.
type LimitOrderCancelBatchReq struct {
	Market   string `url:"market,omitempty" json:"market,omitempty"`
	OrderIDs string `url:"order_ids,omitempty" json:"order_ids,omitempty"` // comma-joined
}

// LimitOrderCancelBatchResp returns the order ids touched.
type LimitOrderCancelBatchResp struct {
	OrderIDs []string `url:"order_ids" json:"order_ids"`
}

// LimitCancelOrderBatch cancels multiple regular orders.
func (c *OrderClient) LimitCancelOrderBatch(ctx context.Context, req LimitOrderCancelBatchReq) (*LimitOrderCancelBatchResp, error) {
	var resp LimitOrderCancelBatchResp
	if err := c.doer.Post(ctx, apiBase+"/order/cancel/batch", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopOrderCancelReq cancels one condition order via POST /order/stop/cancel.
type StopOrderCancelReq struct {
	Market  string `url:"market,omitempty" json:"market,omitempty"`
	OrderID string `url:"order_id,omitempty" json:"order_id,omitempty"`
}

// StopOrderCancelResp returns the affected stop order id (the gateway sends
// the contract_order_id as a 19-digit integer, so int64 is required).
type StopOrderCancelResp struct {
	OrderID int64 `url:"order_id" json:"order_id"`
}

// CancelStopOrder cancels one condition order.
func (c *OrderClient) CancelStopOrder(ctx context.Context, req StopOrderCancelReq) (*StopOrderCancelResp, error) {
	var resp StopOrderCancelResp
	if err := c.doer.Post(ctx, apiBase+"/order/stop/cancel", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopOrderCancelAllReq cancels every condition order in one market.
type StopOrderCancelAllReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// StopOrderCancelAllResp is the empty response of /order/stop/cancel/all.
type StopOrderCancelAllResp struct{}

// CancelAllStopOrder cancels every condition order in one market.
func (c *OrderClient) CancelAllStopOrder(ctx context.Context, req StopOrderCancelAllReq) (*StopOrderCancelAllResp, error) {
	var resp StopOrderCancelAllResp
	if err := c.doer.Post(ctx, apiBase+"/order/stop/cancel/all", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PendingOrderReq filters the open-orders list.
type PendingOrderReq struct {
	Market   string `url:"market,omitempty" json:"market,omitempty"`
	Page     int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// PendingOrderResp is the paginated open-orders response.
type PendingOrderResp struct {
	Records  []OrderItem `url:"records" json:"records"`
	Page     int         `url:"page" json:"page"`
	PageSize int         `url:"page_size" json:"page_size"`
	Count    int         `url:"count" json:"count"`
}

// PendingOrder lists currently open regular orders.
func (c *OrderClient) PendingOrder(ctx context.Context, req PendingOrderReq) (*PendingOrderResp, error) {
	var resp PendingOrderResp
	if err := c.doer.Get(ctx, apiBase+"/order/pending", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FinishedOrderReq filters the historical-orders list.
type FinishedOrderReq struct {
	Market    string `url:"market,omitempty" json:"market,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
	Page      int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize  int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// FinishedOrderResp is the paginated historical-orders response.
type FinishedOrderResp struct {
	Records  []OrderItem `url:"records" json:"records"`
	Page     int         `url:"page" json:"page"`
	PageSize int         `url:"page_size" json:"page_size"`
}

// FinishedOrder lists past regular orders within a time window.
func (c *OrderClient) FinishedOrder(ctx context.Context, req FinishedOrderReq) (*FinishedOrderResp, error) {
	var resp FinishedOrderResp
	if err := c.doer.Get(ctx, apiBase+"/order/finished", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PendingStopOrderReq filters the active-triggers list.
type PendingStopOrderReq struct {
	Market   string `url:"market,omitempty" json:"market,omitempty"`
	Page     int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// PendingStopOrderResp is the paginated active-triggers response.
type PendingStopOrderResp struct {
	Records  []StopOrderItem `url:"records" json:"records"`
	Page     int             `url:"page" json:"page"`
	PageSize int             `url:"page_size" json:"page_size"`
	Count    int             `url:"count" json:"count"`
}

// PendingStopOrder lists currently active condition orders.
func (c *OrderClient) PendingStopOrder(ctx context.Context, req PendingStopOrderReq) (*PendingStopOrderResp, error) {
	var resp PendingStopOrderResp
	if err := c.doer.Get(ctx, apiBase+"/order/stop/pending", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FinishedStopOrderReq filters the historical-triggers list.
type FinishedStopOrderReq struct {
	Market    string `url:"market,omitempty" json:"market,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
	Page      int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize  int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// FinishedStopOrderResp is the paginated historical-triggers response.
type FinishedStopOrderResp struct {
	Records  []StopOrderItem `url:"records" json:"records"`
	Page     int             `url:"page" json:"page"`
	PageSize int             `url:"page_size" json:"page_size"`
	Count    int             `url:"count" json:"count"`
}

// FinishedStopOrder lists past condition orders within a time window.
func (c *OrderClient) FinishedStopOrder(ctx context.Context, req FinishedStopOrderReq) (*FinishedStopOrderResp, error) {
	var resp FinishedStopOrderResp
	if err := c.doer.Get(ctx, apiBase+"/order/stop/finished", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// OrderDetailReq looks up one regular order.
type OrderDetailReq struct {
	Market  string `url:"market,omitempty" json:"market,omitempty"`
	OrderID string `url:"order_id,omitempty" json:"order_id,omitempty"`
}

// OrderDetail fetches one regular order's full record.
func (c *OrderClient) OrderDetail(ctx context.Context, req OrderDetailReq) (*OrderItem, error) {
	var resp OrderItem
	if err := c.doer.Get(ctx, apiBase+"/order/detail", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// OrderDealsReq filters the user's trade history.
type OrderDealsReq struct {
	Market    string `url:"market,omitempty" json:"market,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
	Page      int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize  int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// OrderDealsResp is the paginated trade-history response.
type OrderDealsResp struct {
	Records  []OrderDealItem `url:"records" json:"records"`
	Page     int             `url:"page" json:"page"`
	PageSize int             `url:"page_size" json:"page_size"`
}

// OrderDeals lists the user's executed trades.
func (c *OrderClient) OrderDeals(ctx context.Context, req OrderDealsReq) (*OrderDealsResp, error) {
	var resp OrderDealsResp
	if err := c.doer.Get(ctx, apiBase+"/order/deals", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitOrderEditReq modifies an existing limit order via POST /order/limit/edit.
type LimitOrderEditReq struct {
	Market   string `url:"market,omitempty" json:"market,omitempty"`
	OrderID  string `url:"order_id,omitempty" json:"order_id,omitempty"`
	Price    string `url:"price,omitempty" json:"price,omitempty"`
	Quantity string `url:"quantity,omitempty" json:"quantity,omitempty"`
}

// LimitOrderEditResp wraps the updated OrderItem.
type LimitOrderEditResp struct{ OrderItem }

// EditLimitOrder modifies an existing limit order.
func (c *OrderClient) EditLimitOrder(ctx context.Context, req LimitOrderEditReq) (*LimitOrderEditResp, error) {
	var resp LimitOrderEditResp
	if err := c.doer.Post(ctx, apiBase+"/order/limit/edit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopOrderEditReq modifies a pending condition order via POST /order/stop/edit.
//
// `StopOrderID` is the contract_order_id (a 19-digit string), not the regular
// order_id. The endpoint accepts only position-attached stops
// (StopOrderTypePositionTakeProfit/StopLoss, OrderTakeProfit/StopLoss);
// standalone stops (StopOrderTypeStandalone) are rejected with code=10066
// "stop order type illegal" — to change a standalone stop, cancel and re-submit.
type StopOrderEditReq struct {
	Market        string          `url:"market,omitempty" json:"market,omitempty"`
	StopOrderID   string          `url:"stop_order_id,omitempty" json:"stop_order_id,omitempty"`
	StopPrice     string          `url:"stop_price,omitempty" json:"stop_price,omitempty"`
	StopPriceType StopTriggerType `url:"stop_price_type,omitempty" json:"stop_price_type,omitempty"`
}

// StopOrderEditResp returns the affected stop order id (the gateway sends the
// contract_order_id as a 19-digit integer, so int64 is required).
type StopOrderEditResp struct {
	OrderID int64 `url:"order_id" json:"order_id"`
}

// EditStopOrder modifies a pending condition order. See StopOrderEditReq for
// the standalone-vs-position-attached restriction.
func (c *OrderClient) EditStopOrder(ctx context.Context, req StopOrderEditReq) (*StopOrderEditResp, error) {
	var resp StopOrderEditResp
	if err := c.doer.Post(ctx, apiBase+"/order/stop/edit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopOrderCloseReq attaches SL/TP to a pending order via POST /order/close/stop.
//
// The endpoint requires both SL and TP fields to be sent together; callers
// preserving one side while updating the other must read current values and
// pass both fields.
type StopOrderCloseReq struct {
	Market              string          `url:"market,omitempty" json:"market,omitempty"`
	OrderID             string          `url:"order_id,omitempty" json:"order_id,omitempty"`
	StopLossPrice       string          `url:"stop_loss_price,omitempty" json:"stop_loss_price,omitempty"`
	StopLossPriceType   StopTriggerType `url:"stop_loss_price_type,omitempty" json:"stop_loss_price_type,omitempty"`
	TakeProfitPrice     string          `url:"take_profit_price,omitempty" json:"take_profit_price,omitempty"`
	TakeProfitPriceType StopTriggerType `url:"take_profit_price_type,omitempty" json:"take_profit_price_type,omitempty"`
}

// StopOrderCloseResp is the empty response of /order/close/stop.
type StopOrderCloseResp struct{}

// StopOrderClose attaches SL/TP to a pending order.
func (c *OrderClient) StopOrderClose(ctx context.Context, req StopOrderCloseReq) (*StopOrderCloseResp, error) {
	var resp StopOrderCloseResp
	if err := c.doer.Post(ctx, apiBase+"/order/close/stop", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LimitOrderBatchItemReq is one item in a batch limit-order request.
type LimitOrderBatchItemReq struct {
	Side      Side   `url:"side,omitempty" json:"side,omitempty"`
	Price     string `url:"price,omitempty" json:"price,omitempty"`
	Quantity  string `url:"quantity,omitempty" json:"quantity,omitempty"`
	ClientOID string `url:"client_oid,omitempty" json:"client_oid,omitempty"`
	TIF       TIF    `url:"tif,omitempty" json:"tif,omitempty"`
}

// LimitOrderBatchReq submits multiple limit orders atomically.
type LimitOrderBatchReq struct {
	Market string                   `url:"market,omitempty" json:"market,omitempty"`
	Window int                      `url:"window,omitempty" json:"window,omitempty"`
	Orders []LimitOrderBatchItemReq `url:"orders,omitempty" json:"orders,omitempty"`
}

// LimitOrderBatchResp summarises which client_oids landed.
type LimitOrderBatchResp struct {
	SuccessClientOIDs []string `url:"success_client_oids" json:"success_client_oids"`
	FailedClientOIDs  []string `url:"failed_client_oids" json:"failed_client_oids"`
}

// LimitOrderBatch submits multiple limit orders in one request.
func (c *OrderClient) LimitOrderBatch(ctx context.Context, req LimitOrderBatchReq) (*LimitOrderBatchResp, error) {
	var resp LimitOrderBatchResp
	if err := c.doer.Post(ctx, apiBase+"/order/limit/batch", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
