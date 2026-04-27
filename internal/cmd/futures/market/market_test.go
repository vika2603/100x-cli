package market

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/mocks"
	"github.com/vika2603/100x-cli/internal/output"
	"go.uber.org/mock/gomock"
)

func TestRunKlineEmptyHuman(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/market/kline", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*[]futures.MarketKlineItem) = nil
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &KlineOptions{Symbol: "BTCUSDT", Interval: "1m", Limit: 20, Factory: &factory.Factory{
		Client: futures.NewWithDoer(doer),
		IO:     &output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	}}
	if err := runKline(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No candles found.") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestRunKlineRejectsBadLimitBeforeClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	opts := &KlineOptions{Symbol: "BTCUSDT", Interval: "1m", Limit: 0, Factory: &factory.Factory{
		Client: futures.NewWithDoer(mocks.NewMockDoer(ctrl)),
		IO:     output.New(),
	}}
	err := runKline(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Fatalf("err=%v", err)
	}
}
