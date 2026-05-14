package format

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

func TestPercent(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0.0232976277110443", "+2.33%"},
		{"-0.00039406", "-0.04%"},
		{"0", "+0.00%"},
		{"", "-"},
		{"-", "-"},
		{"  0.5  ", "+50.00%"},
		{"garbage", "garbage"},
	}
	for _, tc := range cases {
		if got := Percent(tc.in); got != tc.want {
			t.Errorf("Percent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
