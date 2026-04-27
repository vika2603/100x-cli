package transport

import (
	"context"
	"net/http"
	"time"
)

// RetryPolicy controls how Client.Get retries transient failures.
// MaxAttempts <= 0 is clamped to 1. MaxElapsed == 0 means no sequence cap.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxElapsed  time.Duration
}

// DefaultRetryPolicy is applied to Client.Get when no policy is set at
// construction or attached to the context.
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts: 3,
	BaseDelay:   200 * time.Millisecond,
	MaxDelay:    1 * time.Second,
}

// NoRetry disables retry for a single call when attached via WithRetryPolicyCtx.
var NoRetry = RetryPolicy{MaxAttempts: 1}

// Option configures a Client at construction.
type Option func(*Client)

// WithRetryPolicy overrides DefaultRetryPolicy for Client.Get calls that
// don't carry a per-call policy on their context.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *Client) { c.retry = p }
}

type retryCtxKey struct{}

// WithRetryPolicyCtx attaches a per-call policy to ctx; it wins over the
// policy set at construction.
func WithRetryPolicyCtx(ctx context.Context, p RetryPolicy) context.Context {
	return context.WithValue(ctx, retryCtxKey{}, p)
}

func retryPolicyFromCtx(ctx context.Context, fallback RetryPolicy) RetryPolicy {
	if v, ok := ctx.Value(retryCtxKey{}).(RetryPolicy); ok {
		return v
	}
	return fallback
}

func retryableStatus(s int) bool {
	switch s {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}
