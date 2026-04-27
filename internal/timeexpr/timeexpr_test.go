package timeexpr

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Run("seconds", func(t *testing.T) {
		got, err := Parse("1777200000", "since")
		if err != nil {
			t.Fatal(err)
		}
		if got != 1777200000 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("milliseconds", func(t *testing.T) {
		got, err := Parse("1777200000000", "since")
		if err != nil {
			t.Fatal(err)
		}
		if got != 1777200000 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("datetime", func(t *testing.T) {
		got, err := Parse("2026-04-26 20:00:00", "since")
		if err != nil {
			t.Fatal(err)
		}
		if got == 0 {
			t.Fatal("got zero")
		}
	})

	t.Run("now minus duration", func(t *testing.T) {
		now := time.Unix(1777200000, 0)
		got, err := parseAt("now-24h", "since", now)
		if err != nil {
			t.Fatal(err)
		}
		if got != 1777113600 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("now plus duration", func(t *testing.T) {
		now := time.Unix(1777200000, 0)
		got, err := parseAt("now+24h", "until", now)
		if err != nil {
			t.Fatal(err)
		}
		if got != 1777286400 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("compound duration", func(t *testing.T) {
		now := time.Unix(1777200000, 0)
		got, err := parseAt("now - 1d2h30m", "since", now)
		if err != nil {
			t.Fatal(err)
		}
		if got != 1777104600 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := Parse("last week", "since"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveRange(t *testing.T) {
	now := time.Unix(1777200000, 0)

	t.Run("since defaults until to now", func(t *testing.T) {
		start, end, err := resolveRangeAt("now-24h", "", now)
		if err != nil {
			t.Fatal(err)
		}
		if start != 1777113600 {
			t.Fatalf("start=%d", start)
		}
		if end != 1777200000 {
			t.Fatalf("end=%d", end)
		}
	})

	t.Run("explicit until wins", func(t *testing.T) {
		start, end, err := resolveRangeAt("now-24h", "now+24h", now)
		if err != nil {
			t.Fatal(err)
		}
		if start != 1777113600 || end != 1777286400 {
			t.Fatalf("start/end=%d/%d", start, end)
		}
	})

	t.Run("rejects since in the future", func(t *testing.T) {
		_, _, err := resolveRangeAt("now+1h", "", now)
		if err == nil || !strings.Contains(err.Error(), "future") {
			t.Fatalf("err=%v want future-rejection", err)
		}
	})

	t.Run("rejects since after until", func(t *testing.T) {
		_, _, err := resolveRangeAt("now", "now-1h", now)
		if err == nil || !strings.Contains(err.Error(), "earlier than --until") {
			t.Fatalf("err=%v want reverse-window rejection", err)
		}
	})

	t.Run("rejects since equal to until", func(t *testing.T) {
		_, _, err := resolveRangeAt("now-1h", "now-1h", now)
		if err == nil || !strings.Contains(err.Error(), "earlier than --until") {
			t.Fatalf("err=%v want reverse-window rejection", err)
		}
	})
}

// resolveRangeAt mirrors ResolveRange but accepts an injected `now` so
// time-window checks remain deterministic.
func resolveRangeAt(since, until string, now time.Time) (int, int, error) {
	nowSec := int(now.Unix())
	start, err := parseAt(since, "since", now)
	if err != nil {
		return 0, 0, err
	}
	if until == "" && since != "" {
		if start > nowSec {
			return 0, 0, fmt.Errorf("--since %q is in the future", since)
		}
		return start, nowSec, nil
	}
	end, err := parseAt(until, "until", now)
	if err != nil {
		return 0, 0, err
	}
	if start != 0 && start > nowSec {
		return 0, 0, fmt.Errorf("--since %q is in the future", since)
	}
	if start != 0 && end != 0 && start >= end {
		return 0, 0, fmt.Errorf("--since %q must be earlier than --until %q", since, until)
	}
	return start, end, nil
}
