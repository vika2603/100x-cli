package futures

import "context"

// SettingClient covers /open/api/v2/setting/*.
type SettingClient struct {
	doer Doer
}

// AdjustMarketPreferenceReq updates the per-market user preference.
//
// The endpoint requires both fields to be sent together; callers preserving
// one field while updating the other must read current values first.
type AdjustMarketPreferenceReq struct {
	Market       string       `url:"market,omitempty" json:"market,omitempty"`
	Leverage     string       `url:"leverage,omitempty" json:"leverage,omitempty"`
	PositionType PositionType `url:"position_type,omitempty" json:"position_type,omitempty"`
}

// AdjustMarketPreferenceResp is the empty response of POST /setting/preference.
type AdjustMarketPreferenceResp struct{}

// AdjustMarketPreference updates the per-market preference.
func (c *SettingClient) AdjustMarketPreference(ctx context.Context, req AdjustMarketPreferenceReq) (*AdjustMarketPreferenceResp, error) {
	var resp AdjustMarketPreferenceResp
	if err := c.doer.Post(ctx, "/open/api/v2/setting/preference", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarketPreferenceReq selects which market's preference to read.
type MarketPreferenceReq struct {
	Market string `url:"market,omitempty" json:"market,omitempty"`
}

// MarketPreference reads the per-market preference.
func (c *SettingClient) MarketPreference(ctx context.Context, req MarketPreferenceReq) (*MarketPreferenceResp, error) {
	var resp MarketPreferenceResp
	if err := c.doer.Get(ctx, "/open/api/v2/setting/preference", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
