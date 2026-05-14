package wire

import "testing"

func TestMarket(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"BTCUSDT", "BTCUSDT"},
		{"btcusdt", "BTCUSDT"},
		{"btc-usdt", "BTCUSDT"},
		{"BTC-USDT", "BTCUSDT"},
		{"BtC-uSDt", "BTCUSDT"},
		{"BTC--USDT", "BTCUSDT"},
	}
	for _, tc := range cases {
		if got := Market(tc.in); got != tc.want {
			t.Errorf("Market(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
