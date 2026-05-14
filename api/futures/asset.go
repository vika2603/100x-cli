package futures

import "context"

// AssetClient covers /open/api/v2/asset/*.
type AssetClient struct {
	doer Doer
}

// AssetQueryReq is the request for the wallet snapshot.
type AssetQueryReq struct{}

// AssetQuery returns the user's current wallet across assets.
func (c *AssetClient) AssetQuery(ctx context.Context, req AssetQueryReq) ([]AssetDetailItem, error) {
	var resp []AssetDetailItem
	if err := c.doer.Get(ctx, apiBase+"/asset/query", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AssetHistoryReq filters the asset-change history.
type AssetHistoryReq struct {
	Asset     string `url:"asset,omitempty" json:"asset,omitempty"`
	Business  string `url:"business,omitempty" json:"business,omitempty"`
	StartTime int    `url:"start_time,omitempty" json:"start_time,omitempty"`
	EndTime   int    `url:"end_time,omitempty" json:"end_time,omitempty"`
	Page      int    `url:"page,omitempty" json:"page,omitempty"`
	PageSize  int    `url:"page_size,omitempty" json:"page_size,omitempty"`
}

// AssetHistoryResp is the paginated asset-history response.
type AssetHistoryResp struct {
	Records  []AssetHistoryItem `url:"records" json:"records"`
	Page     int                `url:"page" json:"page"`
	PageSize int                `url:"page_size" json:"page_size"`
}

// AssetHistory lists asset changes within a time window.
func (c *AssetClient) AssetHistory(ctx context.Context, req AssetHistoryReq) (*AssetHistoryResp, error) {
	var resp AssetHistoryResp
	if err := c.doer.Get(ctx, apiBase+"/asset/history", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
