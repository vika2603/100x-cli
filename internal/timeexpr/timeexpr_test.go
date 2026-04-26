package timeexpr

import (
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
}

func resolveRangeAt(since, until string, now time.Time) (int, int, error) {
	start, err := parseAt(since, "since", now)
	if err != nil {
		return 0, 0, err
	}
	if until == "" && since != "" {
		return start, int(now.Unix()), nil
	}
	end, err := parseAt(until, "until", now)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}
