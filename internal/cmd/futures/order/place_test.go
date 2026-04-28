package order

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/mocks"
	"github.com/vika2603/100x-cli/internal/output"
)

// TestRunPlaceLimit drives verb wiring: flag-bound options → SDK call → renderer.
func TestRunPlaceLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Post(gomock.Any(), "/open/api/v2/order/limit", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, in, out any) error {
			req := in.(futures.LimitOrderReq)
			*out.(*futures.LimitOrderResp) = futures.LimitOrderResp{OrderItem: futures.OrderItem{
				OrderID: 1001, Market: req.Market, Side: req.Side,
				Price: req.Price, Volume: req.Quantity, Status: futures.OrderStatusPending,
			}}
			return nil
		})
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	f := &factory.Factory{
		Client: futures.NewWithDoer(doer),
		IO:     &output.Renderer{Out: stdout, Err: stderr, Format: output.FormatHuman},
		Yes:    true,
	}
	opts := &PlaceOptions{
		Limit: true, Symbol: "BTCUSDT", Side: "buy",
		Price: "70000", Size: "0.1",
		Factory: f,
	}
	if err := runPlace(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if !strings.Contains(got, "BTCUSDT") {
		t.Errorf("stdout missing market: %q", got)
	}
}

// TestRunPlaceMissingType errors out before calling the SDK when neither
// --limit nor --market is set. (Cobra's MarkFlagsOneRequired catches this at
// parse time too; this guards the runPlace direct-call path.)
func TestRunPlaceMissingType(t *testing.T) {
	ctrl := gomock.NewController(t)
	f := &factory.Factory{
		Client: futures.NewWithDoer(mocks.NewMockDoer(ctrl)),
		IO:     output.New(),
	}
	opts := &PlaceOptions{Symbol: "BTC", Side: "buy", Size: "1", Factory: f}
	err := runPlace(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "must set --limit or --market") {
		t.Errorf("unexpected err: %v", err)
	}
}
