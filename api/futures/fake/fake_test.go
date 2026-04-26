package fake

import (
	"context"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
)

// TestFakeImplementsDoer is a compile-time assertion guarded by a runtime
// test so refactors that drift the fake away from futures.Doer fail loudly.
func TestFakeImplementsDoer(_ *testing.T) {
	var _ futures.Doer = New()
}

// TestPlaceListCancelRoundTrip exercises the whole order lifecycle through
// the fake to catch silent state-handling regressions.
func TestPlaceListCancelRoundTrip(t *testing.T) {
	c := futures.NewWithDoer(New())
	ctx := context.Background()

	// Place a limit order.
	resp, err := c.Order.LimitOrder(ctx, futures.LimitOrderReq{
		Market: "BTCUSDT", Side: futures.SideBuy, Price: "70000", Quantity: "0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID == 0 {
		t.Fatal("OrderID = 0")
	}
	if resp.Status != futures.OrderStatusPending {
		t.Fatalf("Status=%v want pending", resp.Status)
	}

	// It should appear in the open list.
	pend, err := c.Order.PendingOrder(ctx, futures.PendingOrderReq{Market: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range pend.Records {
		if o.OrderID == resp.OrderID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("placed order %d not in pending list", resp.OrderID)
	}

	// Cancelling moves it to closed.
	if _, err := c.Order.CancelOrder(ctx, futures.LimitOrderCancelReq{
		Market: "BTCUSDT", OrderID: itoa(resp.OrderID),
	}); err != nil {
		t.Fatal(err)
	}
	fin, err := c.Order.FinishedOrder(ctx, futures.FinishedOrderReq{Market: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, o := range fin.Records {
		if o.OrderID == resp.OrderID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("cancelled order %d not in finished list", resp.OrderID)
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
