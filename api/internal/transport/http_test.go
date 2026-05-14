package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestGetSignsAndDecodes verifies that GET attaches the four auth params and
// that a 0-coded envelope decodes into out.
func TestGetSignsAndDecodes(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "msg": "ok", "data": map[string]string{"hello": "world"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, Credentials{ClientID: "cid", ClientKey: "key"}, nil)
	type req struct {
		Market string `url:"market,omitempty"`
	}
	var out struct {
		Hello string `json:"hello"`
	}
	if err := c.Get(context.Background(), "/p", req{Market: "BTCUSDT"}, &out); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/p" {
		t.Fatalf("path=%q want /p", gotPath)
	}
	for _, k := range []string{"client_id", "nonce", "ts", "sign"} {
		if gotQuery.Get(k) == "" {
			t.Fatalf("missing auth param %s", k)
		}
	}
	// The transport passes the Market field through verbatim. Symbol
	// normalisation is the command layer's responsibility (via format.Market).
	if gotQuery.Get("market") != "BTCUSDT" {
		t.Fatalf("market=%q want BTCUSDT", gotQuery.Get("market"))
	}
	if out.Hello != "world" {
		t.Fatalf("out.Hello=%q want world", out.Hello)
	}
}

// TestPostSendsBodyAndAuth verifies POST puts auth in query and the typed
// struct in the JSON body untouched.
func TestPostSendsBodyAndAuth(t *testing.T) {
	var gotQuery url.Values
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
	}))
	defer srv.Close()

	c := New(srv.URL, Credentials{ClientID: "cid", ClientKey: "key"}, nil)
	type body struct {
		Market   string `json:"market,omitempty"`
		Quantity string `json:"quantity,omitempty"`
	}
	if err := c.Post(context.Background(), "/p", body{Market: "BTCUSDT", Quantity: "1"}, nil); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"client_id", "nonce", "ts", "sign"} {
		if gotQuery.Get(k) == "" {
			t.Fatalf("missing auth param %s in query", k)
		}
	}
	// Auth params must NOT appear in the body — that was the whole point of
	// putting them in the query.
	for _, k := range []string{"client_id", "nonce", "ts", "sign"} {
		if _, ok := gotBody[k]; ok {
			t.Fatalf("auth param %s leaked into body", k)
		}
	}
	if gotBody["market"] != "BTCUSDT" {
		t.Fatalf("body.market=%v want BTCUSDT", gotBody["market"])
	}
}

// TestEnvelopeNonZeroReturnsAPIError ensures non-zero codes surface as APIError.
func TestEnvelopeNonZeroReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":20001,"message":"bad params","data":null}`))
	}))
	defer srv.Close()
	c := New(srv.URL, Credentials{ClientID: "c", ClientKey: "k"}, nil)
	err := c.Get(context.Background(), "/p", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "20001") || !strings.Contains(err.Error(), "bad params") {
		t.Fatalf("unexpected error: %v", err)
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("err type = %T, want *APIError", err)
	}
	if ae.Code != 20001 {
		t.Fatalf("code=%d want 20001", ae.Code)
	}
}
