package protection

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/mocks"
)

const (
	pathOrderDetail   = "/open/api/v2/order/detail"
	pathPendingStops  = "/open/api/v2/order/stop/pending"
	pathStopOrderEdit = "/open/api/v2/order/stop/edit"
	pathStopClose     = "/open/api/v2/order/close/stop"
	pathPositionsList = "/open/api/v2/position/pending"
	pathPositionStop  = "/open/api/v2/position/close/stop"
)

func newClient(t *testing.T) (*futures.Client, *mocks.MockDoer) {
	t.Helper()
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	return futures.NewWithDoer(doer), doer
}

// returnOrderDetail makes the next OrderDetail GET return order.
func returnOrderDetail(doer *mocks.MockDoer, order futures.OrderItem) {
	doer.EXPECT().
		Get(gomock.Any(), pathOrderDetail, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.OrderItem) = order
			return nil
		})
}

// returnPendingStops makes the next PendingStopOrder GET return records.
func returnPendingStops(doer *mocks.MockDoer, records []futures.StopOrderItem) {
	doer.EXPECT().
		Get(gomock.Any(), pathPendingStops, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.PendingStopOrderResp) = futures.PendingStopOrderResp{Records: records}
			return nil
		})
}

func returnPositions(doer *mocks.MockDoer, list []futures.PendingPositionDetail) {
	doer.EXPECT().
		Get(gomock.Any(), pathPositionsList, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*[]futures.PendingPositionDetail) = list
			return nil
		})
}

// expectStopClose captures the StopOrderCloseReq the next POST receives.
func expectStopClose(doer *mocks.MockDoer, captured *futures.StopOrderCloseReq) {
	doer.EXPECT().
		Post(gomock.Any(), pathStopClose, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
			*captured = in.(futures.StopOrderCloseReq)
			return nil
		})
}

func expectStopEdit(doer *mocks.MockDoer, captured *futures.StopOrderEditReq) {
	doer.EXPECT().
		Post(gomock.Any(), pathStopOrderEdit, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
			*captured = in.(futures.StopOrderEditReq)
			return nil
		})
}

func expectPositionStopClose(doer *mocks.MockDoer, captured *futures.StopClosePositionReq) {
	doer.EXPECT().
		Post(gomock.Any(), pathPositionStop, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
			*captured = in.(futures.StopClosePositionReq)
			return nil
		})
}

// TestOrderInspectMergesRecordAndStandaloneTriggers exercises the read path:
// the order's StopLossPrice/TakeProfitPrice on its detail record and any
// matching StopOrder records combine into a single State, with TriggerID
// preserved when a standalone trigger pegs the side.
func TestOrderInspectMergesRecordAndStandaloneTriggers(t *testing.T) {
	c, doer := newClient(t)
	returnOrderDetail(doer, futures.OrderItem{
		OrderID: 1001, PositionID: 7,
		StopLossPrice:   "65000",
		TakeProfitPrice: "75000",
	})
	returnPendingStops(doer, []futures.StopOrderItem{
		{
			ContractOrderID: "abc", OrderID: 1001, PositionID: 7,
			ContractOrderType: futures.StopOrderTypeOrderTakeProfit,
			TriggerType:       futures.StopTriggerTypeMark,
			TriggerPrice:      "76000",
		},
	})

	state, err := Order{Symbol: "BTCUSDT", OrderID: "1001"}.Inspect(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if state.SL.Price != "65000" || state.SL.TriggerID != "" {
		t.Errorf("SL=%+v want price=65000 no TriggerID (no standalone)", state.SL)
	}
	if state.TP.Price != "76000" || state.TP.TriggerID != "abc" || state.TP.PriceType != futures.StopTriggerTypeMark {
		t.Errorf("TP=%+v want price=76000 trigger=abc type=Mark", state.TP)
	}
	if state.CrossOrderConflict {
		t.Error("CrossOrderConflict must be false when only this order's triggers exist")
	}
}

// TestOrderInspectFlagsCrossOrderConflict triggers the conflict signal: a
// pending stop on the same position but a different order id.
func TestOrderInspectFlagsCrossOrderConflict(t *testing.T) {
	c, doer := newClient(t)
	returnOrderDetail(doer, futures.OrderItem{OrderID: 1001, PositionID: 7})
	returnPendingStops(doer, []futures.StopOrderItem{
		{OrderID: 999, PositionID: 7, ContractOrderType: futures.StopOrderTypeOrderStopLoss, TriggerPrice: "60000"},
	})

	state, err := Order{Symbol: "BTCUSDT", OrderID: "1001"}.Inspect(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if !state.CrossOrderConflict {
		t.Fatal("expected CrossOrderConflict=true")
	}
}

// TestOrderApplyColdStartAttachWithSidePreservation verifies the most common
// trigger-attach path: no existing standalone trigger, attach SL while
// preserving an existing TP set on the order record.
func TestOrderApplyColdStartAttachWithSidePreservation(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		TP: Stop{Price: "75000", PriceType: futures.StopTriggerTypeLast},
	}
	want := current
	want.SL = Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast}

	var got futures.StopOrderCloseReq
	expectStopClose(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.StopLossPrice != "60000" {
		t.Errorf("SL=%q want 60000", got.StopLossPrice)
	}
	if got.TakeProfitPrice != "75000" {
		t.Errorf("TP=%q want 75000 (preserved)", got.TakeProfitPrice)
	}
}

// TestOrderApplyClearOtherWipesOpposite verifies that clearing the unchanged
// side omits its fields from the request body.
func TestOrderApplyClearOtherWipesOpposite(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast},
		TP: Stop{Price: "75000", PriceType: futures.StopTriggerTypeLast},
	}
	want := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast},
		// TP intentionally cleared.
	}

	var got futures.StopOrderCloseReq
	expectStopClose(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.TakeProfitPrice != "" {
		t.Errorf("TP=%q want empty (cleared)", got.TakeProfitPrice)
	}
}

// TestOrderApplyEditsExistingStandaloneTrigger checks the edit-in-place
// branch: when only one side moves and that side is already pegged by a
// standalone StopOrder, Apply routes to EditStopOrder keyed on the trigger
// id, not StopOrderClose.
func TestOrderApplyEditsExistingStandaloneTrigger(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast, TriggerID: "abc"},
		TP: Stop{Price: "75000", PriceType: futures.StopTriggerTypeLast, TriggerID: "def"},
	}
	want := current
	want.SL = Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "abc"}

	var got futures.StopOrderEditReq
	expectStopEdit(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.StopOrderID != "abc" {
		t.Errorf("StopOrderID=%q want abc", got.StopOrderID)
	}
	if got.StopPrice != "60000" {
		t.Errorf("StopPrice=%q want 60000", got.StopPrice)
	}
}

// TestOrderApplyEditsBothTriggersIndividually covers case D: when both SL
// and TP exist as standalone triggers and both prices change, Apply must
// emit two EditStopOrder calls keyed by trigger id rather than a single
// /order/close/stop full-body call (the gateway's SL-update branch in
// stop_order_close_logic.go early-returns and would never touch TP).
func TestOrderApplyEditsBothTriggersIndividually(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-1"},
		TP: Stop{Price: "75000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-1"},
	}
	want := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-1"},
		TP: Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-1"},
	}

	var slCall, tpCall futures.StopOrderEditReq
	gomock.InOrder(
		doer.EXPECT().Post(gomock.Any(), pathStopOrderEdit, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
				slCall = in.(futures.StopOrderEditReq)
				return nil
			}),
		doer.EXPECT().Post(gomock.Any(), pathStopOrderEdit, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
				tpCall = in.(futures.StopOrderEditReq)
				return nil
			}),
	)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if slCall.StopOrderID != "sl-1" || slCall.StopPrice != "60000" {
		t.Errorf("SL edit=%+v want trigger=sl-1 price=60000", slCall)
	}
	if tpCall.StopOrderID != "tp-1" || tpCall.StopPrice != "80000" {
		t.Errorf("TP edit=%+v want trigger=tp-1 price=80000", tpCall)
	}
}

// TestOrderApplyMixedFreshAndExistingReturnsExplicitError covers case E:
// when one side already has a standalone trigger and the other side is
// fresh, /order/close/stop's update path early-returns before reaching
// the missing side. Apply must surface that limitation as a clear error
// pointing the user at a manual recovery path instead of silently losing
// one side's protection.
func TestOrderApplyMixedFreshAndExistingReturnsExplicitError(t *testing.T) {
	c, _ := newClient(t)
	current := State{
		SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-abc"},
	}
	want := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-abc"},
		TP: Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast},
	}

	err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want)
	if err == nil {
		t.Fatal("expected explicit error for mixed fresh/existing scenario")
	}
	if !strings.Contains(err.Error(), "sl-abc") {
		t.Errorf("err=%v should reference the existing trigger id sl-abc", err)
	}
	if !strings.Contains(err.Error(), "trigger cancel") {
		t.Errorf("err=%v should suggest the manual cancel workaround", err)
	}
	// No mock expectations: gomock would have failed the test if Apply called the gateway.
}

// TestOrderApplyDualTriggerByPerSide locks in that each side's PriceType
// is sent independently in the cold-start body, so callers can route SL
// on one feed (e.g. MARK) while TP runs on another (e.g. LAST).
func TestOrderApplyDualTriggerByPerSide(t *testing.T) {
	c, doer := newClient(t)
	want := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeMark},
		TP: Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast},
	}

	var got futures.StopOrderCloseReq
	expectStopClose(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, State{}, want); err != nil {
		t.Fatal(err)
	}
	if got.StopLossPriceType != futures.StopTriggerTypeMark {
		t.Errorf("StopLossPriceType=%v want MARK", got.StopLossPriceType)
	}
	if got.TakeProfitPriceType != futures.StopTriggerTypeLast {
		t.Errorf("TakeProfitPriceType=%v want LAST", got.TakeProfitPriceType)
	}
}

// TestOrderApplyAddTPWhenSLStandaloneReturnsExplicitError covers the case
// where the user asks to set TP while SL already exists as a standalone
// trigger (and TP is fresh). The gateway's /order/close/stop SL block
// hits ConditionOrderUpdate-and-return on every call, so the TP block is
// never reached and the request silently loses TP. Apply must surface
// that as an explicit error pointing at the manual recovery path.
func TestOrderApplyAddTPWhenSLStandaloneReturnsExplicitError(t *testing.T) {
	c, _ := newClient(t)
	current := State{
		SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-abc"},
	}
	want := current
	want.TP = Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast}

	err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want)
	if err == nil {
		t.Fatal("expected explicit error when adding TP while SL is standalone")
	}
	if !strings.Contains(err.Error(), "sl-abc") {
		t.Errorf("err=%v should reference SL trigger id sl-abc", err)
	}
	if !strings.Contains(err.Error(), "trigger cancel") {
		t.Errorf("err=%v should suggest the manual cancel workaround", err)
	}
}

// TestOrderApplyAddSLWhenTPStandaloneRunsColdStart confirms the
// reverse direction works without an explicit error: when TP is already
// standalone and the user adds SL, the gateway's SL block takes the
// fresh-entrust path (no early return) and the TP block then runs its
// update path. A single /order/close/stop call carries both fields.
func TestOrderApplyAddSLWhenTPStandaloneRunsColdStart(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		TP: Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-abc"},
	}
	want := current
	want.SL = Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast}

	var got futures.StopOrderCloseReq
	expectStopClose(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.StopLossPrice != "60000" {
		t.Errorf("SL=%q want 60000", got.StopLossPrice)
	}
	if got.TakeProfitPrice != "80000" {
		t.Errorf("TP=%q want 80000 (preserved)", got.TakeProfitPrice)
	}
}

// TestOrderApplyNoOpWhenStateMatches verifies Apply does not call the gateway
// when current and want are equal.
func TestOrderApplyNoOpWhenStateMatches(t *testing.T) {
	c, _ := newClient(t)
	state := State{SL: Stop{Price: "65000", PriceType: futures.StopTriggerTypeLast}}
	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Apply(context.Background(), c, state, state); err != nil {
		t.Fatal(err)
	}
	// No mock expectations set; gomock.NewController would have failed if Apply called the gateway.
}

// TestOrderApplyReattachAfterRebookSendsBothSides covers order-edit's path:
// current is empty (freshly rebooked order), want carries both old SL and
// TP, Apply must send a single StopOrderClose with both fields populated.
func TestOrderApplyReattachAfterRebookSendsBothSides(t *testing.T) {
	c, doer := newClient(t)
	current := State{}
	want := State{
		SL: Stop{Price: "55000", PriceType: futures.StopTriggerTypeLast},
		TP: Stop{Price: "90000", PriceType: futures.StopTriggerTypeLast},
	}

	var got futures.StopOrderCloseReq
	expectStopClose(doer, &got)

	if err := (Order{Symbol: "BTCUSDT", OrderID: "2002"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.OrderID != "2002" || got.StopLossPrice != "55000" || got.TakeProfitPrice != "90000" {
		t.Errorf("body=%+v want OrderID=2002 SL=55000 TP=90000", got)
	}
}

// TestOrderVerifyCatchesMismatch surfaces the "gateway accepted but did not
// apply" case: the order detail comes back with a different price than want.
func TestOrderVerifyCatchesMismatch(t *testing.T) {
	c, doer := newClient(t)
	returnOrderDetail(doer, futures.OrderItem{OrderID: 1001, StopLossPrice: "59999"})

	want := State{SL: Stop{Price: "60000"}}
	err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Verify(context.Background(), c, want)
	if err == nil || !strings.Contains(err.Error(), "59999") {
		t.Fatalf("err=%v want mismatch on SL", err)
	}
}

// TestOrderVerifyIgnoresUnsetSides confirms Verify does not fail when want
// only carries one side: the unset side is not asserted.
func TestOrderVerifyIgnoresUnsetSides(t *testing.T) {
	c, doer := newClient(t)
	returnOrderDetail(doer, futures.OrderItem{OrderID: 1001, StopLossPrice: "60000", TakeProfitPrice: "anything"})

	want := State{SL: Stop{Price: "60000"}} // TP not in want
	if err := (Order{Symbol: "BTCUSDT", OrderID: "1001"}).Verify(context.Background(), c, want); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// TestPositionApplyEditsTriggerInPlace mirrors the order edit-existing-trigger
// branch on the position target.
func TestPositionApplyEditsTriggerInPlace(t *testing.T) {
	c, doer := newClient(t)
	current := State{TP: Stop{Price: "80000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-abc"}}
	want := State{TP: Stop{Price: "85000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-abc"}}

	var got futures.StopOrderEditReq
	expectStopEdit(doer, &got)

	if err := (Position{Symbol: "BTCUSDT", PositionID: "42"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.StopOrderID != "tp-abc" || got.StopPrice != "85000" {
		t.Errorf("got=%+v want trigger=tp-abc price=85000", got)
	}
}

// TestPositionApplyEditsBothTriggersIndividually covers case D on a
// position: when both SL and TP exist as standalone triggers and both
// prices change, Apply emits 2× EditStopOrder keyed by trigger id.
func TestPositionApplyEditsBothTriggersIndividually(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-pos"},
		TP: Stop{Price: "90000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-pos"},
	}
	want := State{
		SL: Stop{Price: "55000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-pos"},
		TP: Stop{Price: "95000", PriceType: futures.StopTriggerTypeLast, TriggerID: "tp-pos"},
	}

	var slCall, tpCall futures.StopOrderEditReq
	gomock.InOrder(
		doer.EXPECT().Post(gomock.Any(), pathStopOrderEdit, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
				slCall = in.(futures.StopOrderEditReq)
				return nil
			}),
		doer.EXPECT().Post(gomock.Any(), pathStopOrderEdit, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, in any, _ any) error {
				tpCall = in.(futures.StopOrderEditReq)
				return nil
			}),
	)

	if err := (Position{Symbol: "BTCUSDT", PositionID: "42"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if slCall.StopOrderID != "sl-pos" || slCall.StopPrice != "55000" {
		t.Errorf("SL edit=%+v want trigger=sl-pos price=55000", slCall)
	}
	if tpCall.StopOrderID != "tp-pos" || tpCall.StopPrice != "95000" {
		t.Errorf("TP edit=%+v want trigger=tp-pos price=95000", tpCall)
	}
}

// TestPositionApplyMixedFreshAndExistingRunsColdStart confirms that a
// position with one standalone side and a fresh other side still routes
// through cold-start /position/close/stop with both fields filled.
// Unlike /order/close/stop, the position handler runs SL/TP in separate
// goroutines and has no early-return bug, so this combination is
// recoverable in a single call.
func TestPositionApplyMixedFreshAndExistingRunsColdStart(t *testing.T) {
	c, doer := newClient(t)
	current := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-pos"},
	}
	want := State{
		SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast, TriggerID: "sl-pos"},
		TP: Stop{Price: "90000", PriceType: futures.StopTriggerTypeLast},
	}

	var got futures.StopClosePositionReq
	expectPositionStopClose(doer, &got)

	if err := (Position{Symbol: "BTCUSDT", PositionID: "42"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.StopLossPrice != "60000" || got.TakeProfitPrice != "90000" {
		t.Errorf("body=%+v want SL=60000 TP=90000", got)
	}
}

// TestPositionApplyColdStartUsesPositionEndpoint verifies that the
// position-target Apply hits the position close/stop endpoint, not the order
// one.
func TestPositionApplyColdStartUsesPositionEndpoint(t *testing.T) {
	c, doer := newClient(t)
	current := State{}
	want := State{SL: Stop{Price: "60000", PriceType: futures.StopTriggerTypeLast}}

	var got futures.StopClosePositionReq
	expectPositionStopClose(doer, &got)

	if err := (Position{Symbol: "BTCUSDT", PositionID: "42"}).Apply(context.Background(), c, current, want); err != nil {
		t.Fatal(err)
	}
	if got.PositionID != "42" || got.StopLossPrice != "60000" {
		t.Errorf("body=%+v want PositionID=42 SL=60000", got)
	}
}

// TestIsAttachedDistinguishesStandalone covers the trigger-edit guard: a
// standalone trigger returns false, an attached trigger returns true, and a
// missing one returns (false, nil).
func TestIsAttachedDistinguishesStandalone(t *testing.T) {
	c, doer := newClient(t)
	stops := []futures.StopOrderItem{
		{ContractOrderID: "stand", ContractOrderType: futures.StopOrderTypeStandalone},
		{ContractOrderID: "ord-sl", ContractOrderType: futures.StopOrderTypeOrderStopLoss},
		{ContractOrderID: "pos-tp", ContractOrderType: futures.StopOrderTypePositionTakeProfit},
	}
	cases := []struct {
		id   string
		want bool
	}{
		{"stand", false},
		{"ord-sl", true},
		{"pos-tp", true},
	}
	for _, tc := range cases {
		returnPendingStops(doer, stops)
		got, err := IsAttached(context.Background(), c, "BTCUSDT", tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("IsAttached(%q)=%v want %v", tc.id, got, tc.want)
		}
	}
}

// TestIsAttachedReturnsFalseWhenMissing covers the not-found case: a trigger
// id that is not currently pending is reported as not attached without error.
func TestIsAttachedReturnsFalseWhenMissing(t *testing.T) {
	c, doer := newClient(t)
	returnPendingStops(doer, []futures.StopOrderItem{})

	got, err := IsAttached(context.Background(), c, "BTCUSDT", "ghost")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("missing trigger must be reported as not attached")
	}
}

// TestPositionInspectFromList verifies that lookupPosition + Inspect read
// PendingPositionDetail.StopLossPrice into the State. Unrelated to mocks
// from above; uses an isolated controller.
func TestPositionInspectFromList(t *testing.T) {
	c, doer := newClient(t)
	returnPositions(doer, []futures.PendingPositionDetail{
		{PositionID: 42, Market: "BTCUSDT", StopLossPrice: "60000", StopLossPriceType: futures.StopTriggerTypeMark},
	})
	returnPendingStops(doer, []futures.StopOrderItem{})

	state, err := Position{Symbol: "BTCUSDT", PositionID: "42"}.Inspect(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if !state.SL.Set() || state.SL.Price != "60000" || state.SL.PriceType != futures.StopTriggerTypeMark {
		t.Errorf("SL=%+v want price=60000 type=Mark", state.SL)
	}
}

// TestOrderInspectPropagatesGetError ensures gateway errors surface unchanged.
func TestOrderInspectPropagatesGetError(t *testing.T) {
	c, doer := newClient(t)
	wantErr := errors.New("boom")
	doer.EXPECT().Get(gomock.Any(), pathOrderDetail, gomock.Any(), gomock.Any()).Return(wantErr)

	_, err := Order{Symbol: "BTCUSDT", OrderID: "1001"}.Inspect(context.Background(), c)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v want %v", err, wantErr)
	}
}
