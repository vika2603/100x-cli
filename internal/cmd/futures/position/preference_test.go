package position

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/mocks"
)

func TestBuildAdjustMarketPreferenceReqMergesPreserved(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	c := futures.NewWithDoer(doer)
	ctx := context.Background()
	current := futures.MarketPreferenceResp{Leverage: "20", PositionType: futures.PositionTypeIsolated}

	t.Run("change leverage only preserves position type", func(t *testing.T) {
		expectPreferenceRead(doer, current)
		req, err := buildAdjustMarketPreferenceReq(ctx, c, mergedPreferenceInput{
			Symbol: "BTCUSDT", Leverage: "50",
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.Leverage != "50" {
			t.Fatalf("leverage=%q want 50", req.Leverage)
		}
		if req.PositionType != futures.PositionTypeIsolated {
			t.Fatalf("position type=%v want isolated (preserved)", req.PositionType)
		}
	})

	t.Run("change position type only preserves leverage", func(t *testing.T) {
		expectPreferenceRead(doer, current)
		req, err := buildAdjustMarketPreferenceReq(ctx, c, mergedPreferenceInput{
			Symbol: "BTCUSDT", PositionType: "CROSS",
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.Leverage != "20" {
			t.Fatalf("leverage=%q want 20 (preserved)", req.Leverage)
		}
		if req.PositionType != futures.PositionTypeCross {
			t.Fatalf("position type=%v want cross", req.PositionType)
		}
	})

	t.Run("both fields skip the read", func(t *testing.T) {
		req, err := buildAdjustMarketPreferenceReq(ctx, c, mergedPreferenceInput{
			Symbol: "BTCUSDT", Leverage: "100", PositionType: "CROSS",
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.Leverage != "100" || req.PositionType != futures.PositionTypeCross {
			t.Fatalf("got %+v", req)
		}
	})

	t.Run("invalid position type errors", func(t *testing.T) {
		expectPreferenceRead(doer, current)
		_, err := buildAdjustMarketPreferenceReq(ctx, c, mergedPreferenceInput{
			Symbol: "BTCUSDT", PositionType: "garbage",
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func expectPreferenceRead(doer *mocks.MockDoer, resp futures.MarketPreferenceResp) {
	doer.EXPECT().
		Get(gomock.Any(), "/open/api/v2/setting/preference", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ any, out any) error {
			*out.(*futures.MarketPreferenceResp) = resp
			return nil
		})
}
