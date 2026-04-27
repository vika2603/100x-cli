package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var fastPolicy = RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond}

func newClient(t *testing.T, srv *httptest.Server, opts ...Option) *Client {
	t.Helper()
	return New(srv.URL, Credentials{ClientID: "cid", ClientKey: "key"}, nil, opts...)
}

func okEnvelope(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]string{"hello": "world"}})
}

func TestGetRetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "boom", http.StatusServiceUnavailable)
			return
		}
		okEnvelope(w)
	}))
	defer srv.Close()

	c := newClient(t, srv, WithRetryPolicy(fastPolicy))
	var out struct{ Hello string }
	if err := c.Get(context.Background(), "/p", nil, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls=%d want 2", got)
	}
	if out.Hello != "world" {
		t.Fatalf("out.Hello=%q want world", out.Hello)
	}
}

func TestPostDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newClient(t, srv, WithRetryPolicy(fastPolicy))
	if err := c.Post(context.Background(), "/p", nil, nil); err == nil {
		t.Fatal("expected error from POST")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls=%d want 1", got)
	}
}

func TestGetSurfacesAPIError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"code":20001,"message":"bad params"}`))
	}))
	defer srv.Close()

	c := newClient(t, srv, WithRetryPolicy(fastPolicy))
	err := c.Get(context.Background(), "/p", nil, nil)
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != 20001 {
		t.Fatalf("expected APIError code=20001, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls=%d want 1", got)
	}
}

func TestGetReSignsEachAttempt(t *testing.T) {
	var (
		mu     sync.Mutex
		nonces []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		nonces = append(nonces, r.URL.Query().Get("nonce"))
		n := len(nonces)
		mu.Unlock()
		if n == 1 {
			http.Error(w, "boom", http.StatusServiceUnavailable)
			return
		}
		okEnvelope(w)
	}))
	defer srv.Close()

	c := newClient(t, srv, WithRetryPolicy(fastPolicy))
	if err := c.Get(context.Background(), "/p", nil, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(nonces) != 2 || nonces[0] == nonces[1] {
		t.Fatalf("nonces=%v want two distinct", nonces)
	}
}

func TestGetCtxOverrideDisablesRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newClient(t, srv, WithRetryPolicy(fastPolicy))
	ctx := WithRetryPolicyCtx(context.Background(), NoRetry)
	if err := c.Get(ctx, "/p", nil, nil); err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls=%d want 1 (NoRetry ctx)", got)
	}
}
