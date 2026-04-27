package shared

import (
	"context"
	"strconv"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/mocks"
	"go.uber.org/mock/gomock"
)

// TestBuildAttachOrderReqPreservesOtherLeg locks in the read-modify-send
// behaviour: setting SL must not clobber the existing TP and vice versa.
func TestBuildAttachOrderReqPreservesOtherLeg(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	c := futures.NewWithDoer(doer)
	ctx := context.Background()
	orderID := int64(1001)
	orderIDText := strconv.FormatInt(orderID, 10)
	current := futures.OrderItem{
		OrderID: orderID, Market: "BTCUSDT", Side: futures.SideBuy,
		Price: "70000", Volume: "1", Status: futures.OrderStatusPending,
		StopLossPrice: "65000", TakeProfitPrice: "75000",
	}

	t.Run("update SL preserves TP", func(t *testing.T) {
		expectOrderDetail(doer, current)
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Symbol: "BTCUSDT", OrderID: orderIDText,
			Leg: LegSL, Price: "60000", PriceType: futures.StopTriggerTypeLast,
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.StopLossPrice != "60000" {
			t.Fatalf("SL=%q want 60000", req.StopLossPrice)
		}
		if req.TakeProfitPrice != "75000" {
			t.Fatalf("TP=%q want preserved 75000", req.TakeProfitPrice)
		}
	})

	t.Run("update TP preserves SL", func(t *testing.T) {
		expectOrderDetail(doer, current)
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Symbol: "BTCUSDT", OrderID: orderIDText,
			Leg: LegTP, Price: "80000", PriceType: futures.StopTriggerTypeMark,
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.TakeProfitPrice != "80000" {
			t.Fatalf("TP=%q want 80000", req.TakeProfitPrice)
		}
		if req.StopLossPrice != "65000" {
			t.Fatalf("SL=%q want preserved 65000", req.StopLossPrice)
		}
	})

	t.Run("ClearOther wipes opposite leg", func(t *testing.T) {
		expectOrderDetail(doer, current)
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Symbol: "BTCUSDT", OrderID: orderIDText,
			Leg: LegSL, Price: "60000", PriceType: futures.StopTriggerTypeLast,
			ClearOther: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.StopLossPrice != "60000" {
			t.Fatalf("SL=%q want 60000", req.StopLossPrice)
		}
		if req.TakeProfitPrice != "" {
			t.Fatalf("TP=%q want empty (cleared)", req.TakeProfitPrice)
		}
	})
}

func expectOrderDetail(doer *mocks.MockDoer, resp futures.OrderItem) {
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/order/detail", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.OrderItem) = resp
			return nil
		})
}

// TestParseLeg covers all valid spellings and rejects junk.
func TestParseLeg(t *testing.T) {
	for _, s := range []string{"SL", "sl", "stop-loss"} {
		if l, err := ParseLeg(s); err != nil || l != LegSL {
			t.Errorf("ParseLeg(%q)=%v,%v want LegSL,nil", s, l, err)
		}
	}
	for _, s := range []string{"TP", "tp", "take-profit"} {
		if l, err := ParseLeg(s); err != nil || l != LegTP {
			t.Errorf("ParseLeg(%q)=%v,%v want LegTP,nil", s, l, err)
		}
	}
	if _, err := ParseLeg("garbage"); err == nil {
		t.Error("ParseLeg(garbage) should error")
	}
}
