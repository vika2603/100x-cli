package trigger

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

func TestRunTriggerListEmptyHuman(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/order/stop/pending", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.PendingStopOrderResp) = futures.PendingStopOrderResp{Records: nil}
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &ListOptions{Symbol: "BTCUSDT", Page: 1, PageSize: 20, Factory: &factory.Factory{
		Client: futures.NewWithDoer(doer),
		IO:     &output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	}}
	if err := runList(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No active triggers found.") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestRunTriggerListRejectsBadPageBeforeClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	opts := &ListOptions{Symbol: "BTCUSDT", Page: 0, PageSize: 20, Factory: &factory.Factory{
		Client: futures.NewWithDoer(mocks.NewMockDoer(ctrl)),
		IO:     output.New(),
	}}
	err := runList(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "--page must be greater than 0") {
		t.Fatalf("err=%v", err)
	}
}
