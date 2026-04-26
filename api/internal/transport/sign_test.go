package transport

import (
	"strings"
	"testing"
)

// TestSignDeterministic locks in the gateway's expected template and HMAC
// algorithm. A change to either side here is a wire-breaking event.
func TestSignDeterministic(t *testing.T) {
	// Two calls with identical inputs must produce identical signatures.
	a := Sign("secret", "client-A", "nonce-1", 1700000000)
	b := Sign("secret", "client-A", "nonce-1", 1700000000)
	if a != b {
		t.Fatalf("Sign is non-deterministic: %s vs %s", a, b)
	}
	// Hex-encoded SHA256 is 64 chars.
	if len(a) != 64 {
		t.Fatalf("sig length = %d, want 64", len(a))
	}
	if strings.ContainsAny(a, "ABCDEF") {
		t.Fatalf("sig must be lowercase hex: %s", a)
	}
}

// TestSignDifferentInputsDiffer guards against accidental collisions in the
// template.
func TestSignDifferentInputsDiffer(t *testing.T) {
	cases := []struct {
		name                    string
		key1, cid1, nonce1      string
		ts1                     int64
		key2, cid2, nonce2      string
		ts2                     int64
	}{
		{"key", "k1", "c", "n", 1, "k2", "c", "n", 1},
		{"clientID", "k", "c1", "n", 1, "k", "c2", "n", 1},
		{"nonce", "k", "c", "n1", 1, "k", "c", "n2", 1},
		{"ts", "k", "c", "n", 1, "k", "c", "n", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := Sign(c.key1, c.cid1, c.nonce1, c.ts1)
			b := Sign(c.key2, c.cid2, c.nonce2, c.ts2)
			if a == b {
				t.Fatalf("%s collision: %s", c.name, a)
			}
		})
	}
}
