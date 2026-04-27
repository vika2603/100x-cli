package futures

import (
	"context"
	"net/http"

	"github.com/vika2603/100x-cli/api/internal/transport"
)

// Doer is the minimal interface every sub-client uses to talk to the gateway.
//
// Implementations:
//   - *transport.Client signs and sends real HTTP requests
//   - generated mocks can be injected in tests
//
// `in` is the typed request struct (or nil); `out` is a pointer the response
// data is decoded into. Auth parameters are added by the implementation and
// never appear in the user's request struct.
type Doer interface {
	Get(ctx context.Context, path string, in, out any) error
	Post(ctx context.Context, path string, in, out any) error
}

// RetryPolicy controls retry on read (GET) requests. See transport.RetryPolicy.
type RetryPolicy = transport.RetryPolicy

// NoRetry disables retry for a single call when attached via WithRetryPolicy.
var NoRetry = transport.NoRetry

// WithRetryPolicy attaches a per-call RetryPolicy to ctx. Writes are not affected.
func WithRetryPolicy(ctx context.Context, p RetryPolicy) context.Context {
	return transport.WithRetryPolicyCtx(ctx, p)
}

// Options configures a futures Client. Retry nil uses transport.DefaultRetryPolicy.
type Options struct {
	Endpoint   string
	ClientID   string
	ClientKey  string
	HTTPClient *http.Client
	Retry      *RetryPolicy
}

// Client is the futures API entry point. Sub-clients group methods by the
// gateway's @server group.
type Client struct {
	Order    *OrderClient
	Position *PositionClient
	Asset    *AssetClient
	Market   *MarketClient
	Setting  *SettingClient
}

// New constructs a Client that signs requests with the given credentials.
func New(opts Options) *Client {
	var trOpts []transport.Option
	if opts.Retry != nil {
		trOpts = append(trOpts, transport.WithRetryPolicy(*opts.Retry))
	}
	tr := transport.New(opts.Endpoint, transport.Credentials{
		ClientID:  opts.ClientID,
		ClientKey: opts.ClientKey,
	}, opts.HTTPClient, trOpts...)
	return NewWithDoer(tr)
}

// NewWithDoer constructs a Client backed by an arbitrary Doer.
func NewWithDoer(d Doer) *Client {
	return &Client{
		Order:    &OrderClient{doer: d},
		Position: &PositionClient{doer: d},
		Asset:    &AssetClient{doer: d},
		Market:   &MarketClient{doer: d},
		Setting:  &SettingClient{doer: d},
	}
}
