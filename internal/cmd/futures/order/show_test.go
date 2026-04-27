package order

import (
	"context"
	"strings"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/mocks"
	"github.com/vika2603/100x-cli/internal/output"
	"go.uber.org/mock/gomock"
)

func TestRunShowRejectsBadOrderIDBeforeClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	opts := &ShowOptions{Symbol: "BTCUSDT", OrderID: "abc", Factory: &factory.Factory{
		Client: futures.NewWithDoer(mocks.NewMockDoer(ctrl)),
		IO:     output.New(),
	}}
	err := runShow(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "order-id must be a positive integer") {
		t.Fatalf("err=%v", err)
	}
}
