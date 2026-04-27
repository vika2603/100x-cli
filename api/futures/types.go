// Package futures is the Go client for the 100x futures-trading open API.
//
// Sub-clients (Order, Position, Asset, Market, Setting) mirror the gateway's
// @server groups. All amounts are strings — the wire format — so callers can
// parse with shopspring/decimal or strconv as needed.
//
// Construct a client with futures.New(Options{...}) for live calls or with
// futures.NewWithDoer(d) to inject a test double.
package futures

// Decimal-valued fields are strings: that is the gateway's wire format.
// Callers parse them with shopspring/decimal or strconv as needed.

// OrderItem is the canonical regular-order shape (limit/market).
type OrderItem struct {
	OrderID         int64        `url:"order_id" json:"order_id"`
	PositionID      int64        `url:"position_id" json:"position_id"`
	Market          string       `url:"market" json:"market"`
	Type            int          `url:"type" json:"type"`
	Side            Side         `url:"side" json:"side"`
	Status          OrderStatus  `url:"status" json:"status"`
	PositionType    PositionType `url:"position_type" json:"position_type"`
	Volume          string       `url:"volume" json:"volume"`
	Left            string       `url:"left" json:"left"`
	Filled          string       `url:"filled" json:"filled"`
	Price           string       `url:"price" json:"price"`
	AvgPrice        string       `url:"avg_price" json:"avg_price"`
	DealStock       string       `url:"deal_stock" json:"deal_stock"`
	DealFee         string       `url:"deal_fee" json:"deal_fee"`
	Leverage        string       `url:"leverage" json:"leverage"`
	StopLossPrice   string       `url:"stop_loss_price" json:"stop_loss_price"`
	TakeProfitPrice string       `url:"take_profit_price" json:"take_profit_price"`
	ClientOID       string       `url:"client_oid" json:"client_oid"`
	Target          int          `url:"target" json:"target"`
	CreateTime      float64      `url:"create_time" json:"create_time"`
	UpdateTime      float64      `url:"update_time" json:"update_time"`
}

// StopOrderItem is the canonical condition-order shape.
type StopOrderItem struct {
	ContractOrderID   string          `url:"contract_order_id" json:"contract_order_id"`
	OrderID           int64           `url:"order_id" json:"order_id"`
	PositionID        int64           `url:"position_id" json:"position_id"`
	Market            string          `url:"market" json:"market"`
	ContractOrderType StopOrderType   `url:"contract_order_type" json:"contract_order_type"`
	TriggerType       StopTriggerType `url:"trigger_type" json:"trigger_type"`
	TriggerPrice      string          `url:"trigger_price" json:"trigger_price"`
	OrderPrice        string          `url:"order_price" json:"order_price"`
	Size              string          `url:"size" json:"size"`
	CutPrice          string          `url:"cut_price" json:"cut_price"`
	Status            StopOrderStatus `url:"status" json:"status"`
	Side              Side            `url:"side" json:"side"`
	PositionType      PositionType    `url:"position_type" json:"position_type"`
	Leverage          string          `url:"leverage" json:"leverage"`
	EntrustID         string          `url:"entrust_id" json:"entrust_id"`
	OrderTime         int             `url:"order_time" json:"order_time"`
	UpdateTime        int             `url:"update_time" json:"update_time"`
}

// OrderDealItem is one filled trade tied to a user's order.
type OrderDealItem struct {
	TradeID      int    `url:"trade_id" json:"trade_id"`
	PositionType int    `url:"position_type" json:"position_type"`
	Market       string `url:"market" json:"market"`
	Time         int    `url:"time" json:"time"`
	Side         Side   `url:"side" json:"side"`
	Leverage     string `url:"leverage" json:"leverage"`
	OrderID      int    `url:"order_id" json:"order_id"`
	Role         int    `url:"role" json:"role"`
	Volume       string `url:"volume" json:"volume"`
	Price        string `url:"price" json:"price"`
	DealFee      string `url:"deal_fee" json:"deal_fee"`
	DealStock    string `url:"deal_stock" json:"deal_stock"`
	FilledType   int    `url:"filled_type" json:"filled_type"`
	TradeType    int    `url:"trade_type" json:"trade_type"`
	DealProfit   string `url:"deal_profit" json:"deal_profit"`
}

// PendingPositionDetail is one row of the open-positions list.
type PendingPositionDetail struct {
	PositionID              int             `url:"position_id" json:"position_id"`
	CreateTime              int             `url:"create_time" json:"create_time"`
	UpdateTime              int             `url:"update_time" json:"update_time"`
	Market                  string          `url:"market" json:"market"`
	Type                    PositionType    `url:"type" json:"type"`
	Side                    Side            `url:"side" json:"side"`
	Volume                  string          `url:"volume" json:"volume"`
	CloseLeft               string          `url:"close_left" json:"close_left"`
	OpenPrice               string          `url:"open_price" json:"open_price"`
	OpenMargin              string          `url:"open_margin" json:"open_margin"`
	MarginAmount            string          `url:"margin_amount" json:"margin_amount"`
	Leverage                string          `url:"leverage" json:"leverage"`
	ProfitUnreal            string          `url:"profit_unreal" json:"profit_unreal"`
	LiqPrice                string          `url:"liq_price" json:"liq_price"`
	MaintenanceMargin       string          `url:"mainten_margin" json:"mainten_margin"`
	MaintenanceMarginAmount string          `url:"mainten_margin_amount" json:"mainten_margin_amount"`
	AdlSort                 int             `url:"adl_sort" json:"adl_sort"`
	Roe                     string          `url:"roe" json:"roe"`
	MarginRatio             string          `url:"margin_ratio" json:"margin_ratio"`
	StopLossPrice           string          `url:"stop_loss_price" json:"stop_loss_price"`
	TakeProfitPrice         string          `url:"take_profit_price" json:"take_profit_price"`
	StopLossPriceType       StopTriggerType `url:"stop_loss_price_type" json:"stop_loss_price_type"`
	TakeProfitPriceType     StopTriggerType `url:"take_profit_price_type" json:"take_profit_price_type"`
	LastPrice               string          `url:"last_price" json:"last_price"`
	SignPrice               string          `url:"sign_price" json:"sign_price"`
	IndexPrice              string          `url:"index_price" json:"index_price"`
}

// FinishedPositionDetail is one row of the closed-positions history.
type FinishedPositionDetail struct {
	PositionID int          `url:"position_id" json:"position_id"`
	CreateTime int          `url:"create_time" json:"create_time"`
	UpdateTime int          `url:"update_time" json:"update_time"`
	Market     string       `url:"market" json:"market"`
	Type       PositionType `url:"type" json:"type"`
	Side       Side         `url:"side" json:"side"`
	OpenPrice  string       `url:"open_price" json:"open_price"`
	ClosePrice string       `url:"close_price" json:"close_price"`
	Leverage   string       `url:"leverage" json:"leverage"`
	FinishType int          `url:"finish_type" json:"finish_type"`
	VolumeMax  string       `url:"volume_max" json:"volume_max"`
	ProfitReal string       `url:"profit_real" json:"profit_real"`
	Roe        string       `url:"roe" json:"roe"`
}

// AssetDetailItem is one slot in the wallet snapshot.
type AssetDetailItem struct {
	Asset        string `url:"asset" json:"asset"`
	Available    string `url:"available" json:"available"`
	Frozen       string `url:"frozen" json:"frozen"`
	Margin       string `url:"margin" json:"margin"`
	BalanceTotal string `url:"balance_total" json:"balance_total"`
	ProfitUnreal string `url:"profit_unreal" json:"profit_unreal"`
	Transfer     string `url:"transfer" json:"transfer"`
	Bonus        string `url:"bonus" json:"bonus"`
}

// AssetHistoryItem is one row of the asset-change history.
type AssetHistoryItem struct {
	Time     int    `url:"time" json:"time"` // milliseconds
	Asset    string `url:"asset" json:"asset"`
	Business string `url:"business" json:"business"`
	Change   string `url:"change" json:"change"`
}

// MarketItem is one tradable instrument descriptor.
type MarketItem struct {
	Type       int        `url:"type" json:"type"`
	Leverages  []string   `url:"leverages" json:"leverages"`
	Name       string     `url:"name" json:"name"`
	Stock      string     `url:"stock" json:"stock"`
	Money      string     `url:"money" json:"money"`
	FeePrec    int        `url:"fee_prec" json:"fee_prec"`
	TickSize   string     `url:"tick_size" json:"tick_size"`
	StockPrec  int        `url:"stock_prec" json:"stock_prec"`
	MoneyPrec  int        `url:"money_prec" json:"money_prec"`
	VolumePrec int        `url:"volume_prec" json:"volume_prec"`
	VolumeMin  string     `url:"volume_min" json:"volume_min"`
	Available  bool       `url:"available" json:"available"`
	Limits     [][]string `url:"limits" json:"limits"`
	Sort       int        `url:"sort" json:"sort"`
	MakerFee   string     `url:"maker_fee" json:"maker_fee"`
	TakerFee   string     `url:"taker_fee" json:"taker_fee"`
}

// MarketStateItem is a per-market ticker snapshot.
//
// Decimal-valued fields are strings (the wire format); callers parse with
// shopspring/decimal or strconv. `Period` is the rolling-window length in
// seconds (e.g. 86400 for 24h). `FundingTime` is seconds-until-next-funding.
type MarketStateItem struct {
	Market             string `url:"market" json:"market"`
	Last               string `url:"last" json:"last"`
	Open               string `url:"open" json:"open"`
	High               string `url:"high" json:"high"`
	Low                string `url:"low" json:"low"`
	Change             string `url:"change" json:"change"`
	Volume             string `url:"volume" json:"volume"`
	Amount             string `url:"amount" json:"amount"`
	Period             int    `url:"period" json:"period"`
	PositionVolume     string `url:"position_volume" json:"position_volume"`
	BuyTotal           string `url:"buy_total" json:"buy_total"`
	SellTotal          string `url:"sell_total" json:"sell_total"`
	IndexPrice         string `url:"index_price" json:"index_price"`
	SignPrice          string `url:"sign_price" json:"sign_price"`
	Insurance          string `url:"insurance" json:"insurance"`
	FundingTime        int64  `url:"funding_time" json:"funding_time"`
	FundingRateLast    string `url:"funding_rate_last" json:"funding_rate_last"`
	FundingRateNext    string `url:"funding_rate_next" json:"funding_rate_next"`
	FundingRatePredict string `url:"funding_rate_predict" json:"funding_rate_predict"`
}

// MarketDepthItem is one level on one side of the book.
type MarketDepthItem struct {
	Price  string `url:"price" json:"price"`
	Volume string `url:"volume" json:"volume"`
}

// MarketDepthResp is the depth-snapshot envelope.
type MarketDepthResp struct {
	IndexPrice string            `url:"index_price" json:"index_price"`
	SignPrice  string            `url:"sign_price" json:"sign_price"`
	Time       int               `url:"time" json:"time"`
	Last       string            `url:"last" json:"last"`
	Asks       []MarketDepthItem `url:"asks" json:"asks"`
	Bids       []MarketDepthItem `url:"bids" json:"bids"`
}

// MarketDealItem is one public trade.
type MarketDealItem struct {
	ID     int    `url:"id" json:"id"`
	Price  string `url:"price" json:"price"`
	Volume string `url:"volume" json:"volume"`
	Type   string `url:"type" json:"type"` // "buy" / "sell"
	Time   int    `url:"time" json:"time"`
}

// MarketKlineItem is one candlestick.
type MarketKlineItem struct {
	Time   int    `url:"time" json:"time"`
	Open   string `url:"open" json:"open"`
	High   string `url:"high" json:"high"`
	Low    string `url:"low" json:"low"`
	Close  string `url:"close" json:"close"`
	Volume string `url:"volume" json:"volume"`
}

// MarketPreferenceResp is the per-market user preference.
type MarketPreferenceResp struct {
	Leverage     string       `url:"leverage" json:"leverage"`
	PositionType PositionType `url:"position_type" json:"position_type"`
}

// PositionAdjustableMarginResp is the response of GET /position/margin.
//
// `Amount` is the position size; `MarginAmount` is the margin currently locked
// against it; `Available` is the wallet balance free to be added; and
// `MaxRemovableMargin` is the upper bound for a remove-margin call. Cross
// positions return `MaxRemovableMargin = "0"`.
type PositionAdjustableMarginResp struct {
	Leverage           string `url:"leverage" json:"leverage"`
	Amount             string `url:"amount" json:"amount"`
	MarginAmount       string `url:"margin_amount" json:"margin_amount"`
	Available          string `url:"available" json:"available"`
	MaxRemovableMargin string `url:"max_removable_margin" json:"max_removable_margin"`
}
