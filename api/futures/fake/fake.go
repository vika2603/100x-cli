// Package fake provides an in-memory Doer for tests and local development.
//
// Each endpoint returns a canned response shape; mutating endpoints update
// shared state held on the Doer so reads see prior writes.
package fake

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/vika2603/100x-cli/api/futures"
)

// Doer is an in-memory implementation of futures.Doer.
//
// Construct via New(); pass the result to futures.NewWithDoer.
type Doer struct {
	mu      sync.Mutex
	orders  map[int64]futures.OrderItem
	stops   map[int64]futures.StopOrderItem
	posByID map[int64]futures.PendingPositionDetail
	pref    map[string]futures.MarketPreferenceResp
	balance []futures.AssetDetailItem
	nextID  atomic.Int64
}

// New constructs an empty in-memory Doer.
func New() *Doer {
	d := &Doer{
		orders:  map[int64]futures.OrderItem{},
		stops:   map[int64]futures.StopOrderItem{},
		posByID: map[int64]futures.PendingPositionDetail{},
		pref:    map[string]futures.MarketPreferenceResp{},
		balance: []futures.AssetDetailItem{
			{Asset: "USDT", Available: "10000", BalanceTotal: "10000", Transfer: "10000"},
		},
	}
	d.nextID.Store(1000)
	return d
}

// Get satisfies futures.Doer for read endpoints.
func (d *Doer) Get(_ context.Context, path string, in, out any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch path {
	case "/open/api/v2/order/pending":
		return assignOut(out, futures.PendingOrderResp{
			Records:  collectOrders(d.orders, futures.OrderStatusPending, futures.OrderStatusPartial),
			Page:     1,
			PageSize: 50,
			Count:    len(d.orders),
		})

	case "/open/api/v2/order/finished":
		return assignOut(out, futures.FinishedOrderResp{
			Records:  collectOrders(d.orders, futures.OrderStatusFilled, futures.OrderStatusCanceled, futures.OrderStatusPartialCanceled),
			Page:     1,
			PageSize: 50,
		})

	case "/open/api/v2/order/stop/pending":
		return assignOut(out, futures.PendingStopOrderResp{
			Records:  collectStops(d.stops, futures.StopOrderStatusUnactivated, futures.StopOrderStatusUntriggered),
			Page:     1,
			PageSize: 50,
			Count:    len(d.stops),
		})

	case "/open/api/v2/order/stop/finished":
		return assignOut(out, futures.FinishedStopOrderResp{
			Records:  collectStops(d.stops, futures.StopOrderStatusSuccess, futures.StopOrderStatusCanceled, futures.StopOrderStatusFailed),
			Page:     1,
			PageSize: 50,
		})

	case "/open/api/v2/order/detail":
		req := in.(futures.OrderDetailReq)
		id, _ := strconv.ParseInt(req.OrderID, 10, 64)
		o, ok := d.orders[id]
		if !ok {
			return fmt.Errorf("fake: order %d not found", id)
		}
		return assignOut(out, o)

	case "/open/api/v2/position/pending":
		out2 := make([]futures.PendingPositionDetail, 0, len(d.posByID))
		for _, p := range d.posByID {
			out2 = append(out2, p)
		}
		return assignOut(out, out2)

	case "/open/api/v2/position/margin":
		return assignOut(out, futures.PositionAdjustableMarginResp{Leverage: "10", Amount: "0"})

	case "/open/api/v2/position/history":
		return assignOut(out, futures.PositionHistoryResp{Records: []futures.FinishedPositionDetail{}, Page: 1, PageSize: 50})

	case "/open/api/v2/setting/preference":
		req := in.(futures.MarketPreferenceReq)
		p, ok := d.pref[req.Market]
		if !ok {
			p = futures.MarketPreferenceResp{Leverage: "10", PositionType: futures.PositionTypeCross}
		}
		return assignOut(out, p)

	case "/open/api/v2/asset/query":
		return assignOut(out, d.balance)

	case "/open/api/v2/asset/history":
		return assignOut(out, futures.AssetHistoryResp{Records: []futures.AssetHistoryItem{}, Page: 1, PageSize: 50})

	case "/open/api/v2/market/list":
		return assignOut(out, []futures.MarketItem{
			{Name: "BTCUSDT", Stock: "BTC", Money: "USDT", Available: true, Leverages: []string{"1", "10", "50", "100"}, TickSize: "0.1"},
		})

	case "/open/api/v2/market/state":
		return assignOut(out, futures.MarketStateItem{Market: "BTCUSDT"})

	case "/open/api/v2/market/state/all":
		return assignOut(out, []futures.MarketStateItem{{Market: "BTCUSDT"}})

	case "/open/api/v2/market/depth":
		return assignOut(out, futures.MarketDepthResp{
			Asks: []futures.MarketDepthItem{{Price: "70010", Volume: "0.5"}},
			Bids: []futures.MarketDepthItem{{Price: "69990", Volume: "0.5"}},
			Last: "70000",
		})

	case "/open/api/v2/market/deals":
		return assignOut(out, []futures.MarketDealItem{})

	case "/open/api/v2/market/kline":
		return assignOut(out, []futures.MarketKlineItem{})

	case "/open/api/v2/order/deals":
		return assignOut(out, futures.OrderDealsResp{Records: []futures.OrderDealItem{}, Page: 1, PageSize: 50})
	}

	return fmt.Errorf("fake: unhandled GET %s", path)
}

// Post satisfies futures.Doer for write endpoints.
func (d *Doer) Post(_ context.Context, path string, in, out any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch path {
	case "/open/api/v2/order/limit":
		req := in.(futures.LimitOrderReq)
		id := d.nextID.Add(1)
		o := futures.OrderItem{
			OrderID: id, Market: req.Market, Side: req.Side,
			Price: req.Price, Volume: req.Quantity, Left: req.Quantity,
			Status: futures.OrderStatusPending, ClientOID: req.ClientOID,
			StopLossPrice: req.StopLossPrice, TakeProfitPrice: req.TakeProfitPrice,
		}
		d.orders[id] = o
		return assignOut(out, futures.LimitOrderResp{OrderItem: o})

	case "/open/api/v2/order/market":
		req := in.(futures.MarketOrderReq)
		id := d.nextID.Add(1)
		o := futures.OrderItem{
			OrderID: id, Market: req.Market, Side: req.Side,
			Volume: req.Quantity, Filled: req.Quantity,
			Status: futures.OrderStatusFilled, ClientOID: req.ClientOID,
		}
		d.orders[id] = o
		return assignOut(out, futures.MarketOrderResp{OrderItem: o})

	case "/open/api/v2/order/stop":
		req := in.(futures.StopOrderReq)
		id := d.nextID.Add(1)
		s := futures.StopOrderItem{
			OrderID: id, Market: req.Market, Side: req.Side,
			TriggerPrice: req.StopPrice, OrderPrice: req.OrderPrice,
			Size: req.Quantity, Status: futures.StopOrderStatusUntriggered,
			ContractOrderType: futures.StopOrderTypeStandalone,
			TriggerType:       req.StopPriceType,
		}
		d.stops[id] = s
		return assignOut(out, futures.StopOrderResp{})

	case "/open/api/v2/order/cancel":
		req := in.(futures.LimitOrderCancelReq)
		id, _ := strconv.ParseInt(req.OrderID, 10, 64)
		o, ok := d.orders[id]
		if !ok {
			return fmt.Errorf("fake: order %d not found", id)
		}
		o.Status = futures.OrderStatusCanceled
		d.orders[id] = o
		return assignOut(out, futures.LimitOrderCancelResp{OrderItem: o})

	case "/open/api/v2/order/cancel/all":
		req := in.(futures.LimitOrderCancelAllReq)
		for id, o := range d.orders {
			if o.Market == req.Market && o.Status <= futures.OrderStatusPartial {
				o.Status = futures.OrderStatusCanceled
				d.orders[id] = o
			}
		}
		return assignOut(out, futures.LimitOrderCancelAllResp{})

	case "/open/api/v2/order/cancel/batch":
		return assignOut(out, futures.LimitOrderCancelBatchResp{})

	case "/open/api/v2/order/stop/cancel":
		req := in.(futures.StopOrderCancelReq)
		id, _ := strconv.ParseInt(req.OrderID, 10, 64)
		s, ok := d.stops[id]
		if !ok {
			return fmt.Errorf("fake: stop %d not found", id)
		}
		s.Status = futures.StopOrderStatusCanceled
		d.stops[id] = s
		return assignOut(out, futures.StopOrderCancelResp{OrderID: id})

	case "/open/api/v2/order/stop/cancel/all":
		req := in.(futures.StopOrderCancelAllReq)
		for id, s := range d.stops {
			if s.Market == req.Market && s.Status <= futures.StopOrderStatusUntriggered {
				s.Status = futures.StopOrderStatusCanceled
				d.stops[id] = s
			}
		}
		return assignOut(out, futures.StopOrderCancelAllResp{})

	case "/open/api/v2/order/limit/edit":
		req := in.(futures.LimitOrderEditReq)
		id, _ := strconv.ParseInt(req.OrderID, 10, 64)
		o, ok := d.orders[id]
		if !ok {
			return fmt.Errorf("fake: order %d not found", id)
		}
		if req.Price != "" {
			o.Price = req.Price
		}
		if req.Quantity != "" {
			o.Volume = req.Quantity
		}
		d.orders[id] = o
		return assignOut(out, futures.LimitOrderEditResp{OrderItem: o})

	case "/open/api/v2/order/stop/edit":
		req := in.(futures.StopOrderEditReq)
		id, _ := strconv.ParseInt(req.StopOrderID, 10, 64)
		s, ok := d.stops[id]
		if !ok {
			return fmt.Errorf("fake: stop %d not found", id)
		}
		if req.StopPrice != "" {
			s.TriggerPrice = req.StopPrice
		}
		if req.StopPriceType != 0 {
			s.TriggerType = req.StopPriceType
		}
		d.stops[id] = s
		return assignOut(out, futures.StopOrderEditResp{OrderID: id})

	case "/open/api/v2/order/close/stop":
		return assignOut(out, futures.StopOrderCloseResp{})

	case "/open/api/v2/order/limit/batch":
		return assignOut(out, futures.LimitOrderBatchResp{})

	case "/open/api/v2/order/report":
		return assignOut(out, futures.ReportOrderResp{})

	case "/open/api/v2/position/margin":
		return assignOut(out, futures.PendingPositionDetail{})

	case "/open/api/v2/position/close/limit", "/open/api/v2/position/close/market",
		"/open/api/v2/position/add/limit", "/open/api/v2/position/add/market":
		id := d.nextID.Add(1)
		return assignOut(out, futures.OrderItem{OrderID: id, Status: futures.OrderStatusPending})

	case "/open/api/v2/position/close/stop":
		return assignOut(out, futures.StopClosePositionResp{})

	case "/open/api/v2/setting/preference":
		req := in.(futures.AdjustMarketPreferenceReq)
		d.pref[req.Market] = futures.MarketPreferenceResp{Leverage: req.Leverage, PositionType: req.PositionType}
		return assignOut(out, futures.AdjustMarketPreferenceResp{})
	}

	return fmt.Errorf("fake: unhandled POST %s", path)
}

func collectOrders(m map[int64]futures.OrderItem, want ...futures.OrderStatus) []futures.OrderItem {
	out := make([]futures.OrderItem, 0, len(m))
	for _, o := range m {
		for _, s := range want {
			if o.Status == s {
				out = append(out, o)
				break
			}
		}
	}
	return out
}

func collectStops(m map[int64]futures.StopOrderItem, want ...futures.StopOrderStatus) []futures.StopOrderItem {
	out := make([]futures.StopOrderItem, 0, len(m))
	for _, s := range m {
		for _, st := range want {
			if s.Status == st {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// assignOut writes src into dst when dst is a non-nil pointer to a
// concrete-type matching destination. The double type-assert pattern lets the
// caller pass typed pointers without us going through json.
func assignOut(dst, src any) error {
	if dst == nil {
		return nil
	}
	switch d := dst.(type) {
	case *futures.OrderItem:
		*d = src.(futures.OrderItem)
	case *futures.LimitOrderResp:
		*d = src.(futures.LimitOrderResp)
	case *futures.MarketOrderResp:
		*d = src.(futures.MarketOrderResp)
	case *futures.StopOrderResp:
		*d = src.(futures.StopOrderResp)
	case *futures.LimitOrderCancelResp:
		*d = src.(futures.LimitOrderCancelResp)
	case *futures.LimitOrderCancelAllResp:
		*d = src.(futures.LimitOrderCancelAllResp)
	case *futures.LimitOrderCancelBatchResp:
		*d = src.(futures.LimitOrderCancelBatchResp)
	case *futures.StopOrderCancelResp:
		*d = src.(futures.StopOrderCancelResp)
	case *futures.StopOrderCancelAllResp:
		*d = src.(futures.StopOrderCancelAllResp)
	case *futures.LimitOrderEditResp:
		*d = src.(futures.LimitOrderEditResp)
	case *futures.StopOrderEditResp:
		*d = src.(futures.StopOrderEditResp)
	case *futures.StopOrderCloseResp:
		*d = src.(futures.StopOrderCloseResp)
	case *futures.LimitOrderBatchResp:
		*d = src.(futures.LimitOrderBatchResp)
	case *futures.ReportOrderResp:
		*d = src.(futures.ReportOrderResp)
	case *futures.PendingOrderResp:
		*d = src.(futures.PendingOrderResp)
	case *futures.FinishedOrderResp:
		*d = src.(futures.FinishedOrderResp)
	case *futures.PendingStopOrderResp:
		*d = src.(futures.PendingStopOrderResp)
	case *futures.FinishedStopOrderResp:
		*d = src.(futures.FinishedStopOrderResp)
	case *futures.OrderDealsResp:
		*d = src.(futures.OrderDealsResp)
	case *futures.PendingPositionDetail:
		*d = src.(futures.PendingPositionDetail)
	case *[]futures.PendingPositionDetail:
		*d = src.([]futures.PendingPositionDetail)
	case *futures.PositionAdjustableMarginResp:
		*d = src.(futures.PositionAdjustableMarginResp)
	case *futures.PositionHistoryResp:
		*d = src.(futures.PositionHistoryResp)
	case *futures.LimitClosePositionResp:
		*d = src.(futures.LimitClosePositionResp)
	case *futures.MarketClosePositionResp:
		*d = src.(futures.MarketClosePositionResp)
	case *futures.StopClosePositionResp:
		*d = src.(futures.StopClosePositionResp)
	case *futures.LimitAddPositionResp:
		*d = src.(futures.LimitAddPositionResp)
	case *futures.MarketAddPositionResp:
		*d = src.(futures.MarketAddPositionResp)
	case *[]futures.AssetDetailItem:
		*d = src.([]futures.AssetDetailItem)
	case *futures.AssetHistoryResp:
		*d = src.(futures.AssetHistoryResp)
	case *[]futures.MarketItem:
		*d = src.([]futures.MarketItem)
	case *futures.MarketStateItem:
		*d = src.(futures.MarketStateItem)
	case *[]futures.MarketStateItem:
		*d = src.([]futures.MarketStateItem)
	case *futures.MarketDepthResp:
		*d = src.(futures.MarketDepthResp)
	case *[]futures.MarketDealItem:
		*d = src.([]futures.MarketDealItem)
	case *[]futures.MarketKlineItem:
		*d = src.([]futures.MarketKlineItem)
	case *futures.MarketPreferenceResp:
		*d = src.(futures.MarketPreferenceResp)
	case *futures.AdjustMarketPreferenceResp:
		*d = src.(futures.AdjustMarketPreferenceResp)
	default:
		return fmt.Errorf("fake: unhandled output type %T", dst)
	}
	return nil
}
