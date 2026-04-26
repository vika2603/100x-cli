package shared

import (
	"context"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/api/futures/fake"
)

// TestBuildAttachOrderReqPreservesOtherLeg locks in the read-modify-send
// behaviour: setting SL must not clobber the existing TP and vice versa.
func TestBuildAttachOrderReqPreservesOtherLeg(t *testing.T) {
	d := fake.New()
	c := futures.NewWithDoer(d)
	ctx := context.Background()

	// Seed an order with both legs already set on the gateway.
	resp, err := c.Order.LimitOrder(ctx, futures.LimitOrderReq{
		Market:          "BTCUSDT",
		Side:            futures.SideBuy,
		Price:           "70000",
		Quantity:        "1",
		StopLossPrice:   "65000",
		TakeProfitPrice: "75000",
	})
	if err != nil {
		t.Fatal(err)
	}
	orderID := resp.OrderID

	t.Run("update SL preserves TP", func(t *testing.T) {
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Market: "BTCUSDT", OrderID: itoa(orderID),
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
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Market: "BTCUSDT", OrderID: itoa(orderID),
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
		req, err := BuildAttachOrderReq(ctx, c, AttachOrderInput{
			Market: "BTCUSDT", OrderID: itoa(orderID),
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

func itoa(i int64) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
