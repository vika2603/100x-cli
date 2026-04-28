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

func TestRunListOpenEmptyHuman(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/order/pending", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.PendingOrderResp) = futures.PendingOrderResp{Records: nil}
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &ListOptions{Symbol: "BTCUSDT", Page: 1, PageSize: 20, Factory: factory.NewForTest(
		futures.NewWithDoer(doer),
		&output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	)}
	if err := runList(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No open orders found.") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestRunListOpenEmptyJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/order/pending", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.PendingOrderResp) = futures.PendingOrderResp{Records: nil}
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &ListOptions{Symbol: "BTCUSDT", Page: 1, PageSize: 20, Factory: factory.NewForTest(
		futures.NewWithDoer(doer),
		&output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatJSON},
	)}
	if err := runList(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "[]" {
		t.Fatalf("stdout=%q", got)
	}
}

func TestRunListFinishedUsesFinishedHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/order/finished", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.FinishedOrderResp) = futures.FinishedOrderResp{Records: []futures.OrderItem{{
				OrderID: 1, Market: "BTCUSDT", Type: 1, Side: futures.SideBuy, Price: "60000", Volume: "0.001",
				Filled: "0.001", UpdateTime: 1800000000,
			}}}
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &ListOptions{Symbol: "BTCUSDT", Finished: true, Page: 1, PageSize: 20, Factory: factory.NewForTest(
		futures.NewWithDoer(doer),
		&output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	)}
	if err := runList(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Finished") {
		t.Fatalf("stdout missing Finished header: %q", got)
	}
	if !strings.Contains(got, "Type") || !strings.Contains(got, "LIMIT") {
		t.Fatalf("stdout missing order type: %q", got)
	}
	if strings.Contains(got, "Updated") {
		t.Fatalf("stdout should not use Updated header: %q", got)
	}
}

func TestRunListRejectsBadPageSizeBeforeClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	opts := &ListOptions{Page: 1, PageSize: 0, Factory: factory.NewForTest(
		futures.NewWithDoer(mocks.NewMockDoer(ctrl)),
		nil,
	)}
	err := runList(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "--page-size must be greater than 0") {
		t.Fatalf("err=%v", err)
	}
}
