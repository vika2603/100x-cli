package order

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/api/futures/fake"
	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/output"
)

// TestRunPlaceLimit drives runPlace end-to-end against the in-memory fake.
//
// This is the bell-curve test for verb wiring: flag-bound options →
// SDK call → renderer. If you broke any of the three, this fails.
func TestRunPlaceLimit(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	f := &factory.Factory{
		Client: futures.NewWithDoer(fake.New()),
		IO:     &output.Renderer{Out: stdout, Err: stderr, Format: output.FormatHuman},
	}
	opts := &PlaceOptions{
		Type: "limit", Market: "BTCUSDT", Side: "buy",
		Price: "70000", Quantity: "0.1",
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

// TestRunPlaceMarketDryRun verifies --dry-run does not call the SDK.
func TestRunPlaceMarketDryRun(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	f := &factory.Factory{
		Client: futures.NewWithDoer(fake.New()),
		IO:     &output.Renderer{Out: stdout, Err: stderr, Format: output.FormatHuman},
		DryRun: true,
	}
	opts := &PlaceOptions{
		Type: "market", Market: "BTCUSDT", Side: "sell", Quantity: "0.1",
		Factory: f,
	}
	if err := runPlace(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Errorf("dry-run wrote to stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "dry-run") {
		t.Errorf("stderr missing dry-run notice: %q", stderr.String())
	}
}

// TestRunPlaceUnknownType errors out before calling the SDK.
func TestRunPlaceUnknownType(t *testing.T) {
	f := &factory.Factory{
		Client: futures.NewWithDoer(fake.New()),
		IO:     output.New(),
	}
	opts := &PlaceOptions{Type: "stop", Market: "BTC", Side: "buy", Quantity: "1", Factory: f}
	err := runPlace(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "unknown --type") {
		t.Errorf("unexpected err: %v", err)
	}
}
