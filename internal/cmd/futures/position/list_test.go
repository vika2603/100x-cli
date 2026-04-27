package position

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

func TestRunPositionListEmptyHuman(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/position/pending", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*[]futures.PendingPositionDetail) = nil
			return nil
		})
	stdout := &bytes.Buffer{}
	opts := &ListOptions{Symbol: "BTCUSDT", Factory: &factory.Factory{
		Client: futures.NewWithDoer(doer),
		IO:     &output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	}}
	if err := runList(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No open positions found.") {
		t.Fatalf("stdout=%q", got)
	}
}
