package shared

import (
	"context"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/api/futures/fake"
)

// TestBuildAdjustMarketPreferenceReqMergesPreserved verifies the set-both
// compensation: a partial CLI update reads current and merges before writing.
func TestBuildAdjustMarketPreferenceReqMergesPreserved(t *testing.T) {
	d := fake.New()
	c := futures.NewWithDoer(d)
	ctx := context.Background()

	// Seed gateway state via the fake's POST path.
	if _, err := c.Setting.AdjustMarketPreference(ctx, futures.AdjustMarketPreferenceReq{
		Market: "BTCUSDT", Leverage: "20", PositionType: futures.PositionTypeIsolated,
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("change leverage only preserves position type", func(t *testing.T) {
		req, err := BuildAdjustMarketPreferenceReq(ctx, c, MergedPreferenceInput{
			Market: "BTCUSDT", Leverage: "50",
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
		req, err := BuildAdjustMarketPreferenceReq(ctx, c, MergedPreferenceInput{
			Market: "BTCUSDT", PositionType: "cross",
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
		req, err := BuildAdjustMarketPreferenceReq(ctx, c, MergedPreferenceInput{
			Market: "BTCUSDT", Leverage: "100", PositionType: "cross",
		})
		if err != nil {
			t.Fatal(err)
		}
		if req.Leverage != "100" || req.PositionType != futures.PositionTypeCross {
			t.Fatalf("got %+v", req)
		}
	})

	t.Run("invalid position type errors", func(t *testing.T) {
		_, err := BuildAdjustMarketPreferenceReq(ctx, c, MergedPreferenceInput{
			Market: "BTCUSDT", PositionType: "garbage",
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseMarginAction(t *testing.T) {
	if a, err := ParseMarginAction("add"); err != nil || a != futures.MarginActionAdd {
		t.Errorf("add => %v,%v", a, err)
	}
	if a, err := ParseMarginAction("remove"); err != nil || a != futures.MarginActionRemove {
		t.Errorf("remove => %v,%v", a, err)
	}
	if _, err := ParseMarginAction("garbage"); err == nil {
		t.Error("garbage should error")
	}
}
