package futures

import "context"

// MarketClient covers public market data under /open/api/v2/market/*.
type MarketClient struct {
	doer Doer
}

// MarketListReq is the request for the instruments list.
type MarketListReq struct{}

// MarketList lists every tradable instrument.
func (c *MarketClient) MarketList(ctx context.Context, req MarketListReq) ([]MarketItem, error) {
	var resp []MarketItem
	if err := c.doer.Get(ctx, "/open/api/v2/market/list", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// MarketStateAllReq is the request for the all-markets ticker.
type MarketStateAllReq struct{}

// MarketStateAll returns ticker snapshots for every market.
func (c *MarketClient) MarketStateAll(ctx context.Context, req MarketStateAllReq) ([]MarketStateItem, error) {
	var resp []MarketStateItem
	if err := c.doer.Get(ctx, "/open/api/v2/market/state/all", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// MarketStateReq selects one market for the ticker.
type MarketStateReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// MarketState returns one market's ticker snapshot.
func (c *MarketClient) MarketState(ctx context.Context, req MarketStateReq) (*MarketStateItem, error) {
	var resp MarketStateItem
	if err := c.doer.Get(ctx, "/open/api/v2/market/state", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketDepthReq selects market and merge step.
type MarketDepthReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
	Merge  string `url:"merge,omitempty" json:"merge,omitempty"`
}

// MarketDepth returns the order-book snapshot.
func (c *MarketClient) MarketDepth(ctx context.Context, req MarketDepthReq) (*MarketDepthResp, error) {
	var resp MarketDepthResp
	if err := c.doer.Get(ctx, "/open/api/v2/market/depth", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketDealsReq selects the market for the public-trade feed.
type MarketDealsReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// MarketDeals returns the latest public trades for one market.
func (c *MarketClient) MarketDeals(ctx context.Context, req MarketDealsReq) ([]MarketDealItem, error) {
	var resp []MarketDealItem
	if err := c.doer.Get(ctx, "/open/api/v2/market/deals", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// MarketKlineReq selects market, candle type, and time window.
type MarketKlineReq struct {
	Market    string `url:"market,omitempty" json:"market,omitempty"`
	Type      string `url:"type,omitempty" json:"type,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
}

// MarketKline returns candlestick history.
func (c *MarketClient) MarketKline(ctx context.Context, req MarketKlineReq) ([]MarketKlineItem, error) {
	var resp []MarketKlineItem
	if err := c.doer.Get(ctx, "/open/api/v2/market/kline", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
