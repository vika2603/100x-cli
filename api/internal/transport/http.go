package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

const defaultTimeout = 15 * time.Second

// Credentials carries the signing identity. Both fields are required.
type Credentials struct {
	ClientID  string
	ClientKey string
}

// Client is the signed HTTP transport for the 100x open API.
//
// It wraps an *req.Client; signing parameters are added to the URL query string
// in a BeforeRequest hook so business request bodies stay clean.
type Client struct {
	Endpoint string
	Creds    Credentials
	r        *req.Client
}

// New constructs a transport client with sane defaults.
//
// `httpClient` is honoured for backward compatibility: when non-nil its
// Timeout and Transport are copied onto the underlying req.Client.
func New(endpoint string, creds Credentials, httpClient *http.Client) *Client {
	r := req.C().
		SetBaseURL(strings.TrimRight(endpoint, "/")).
		SetTimeout(defaultTimeout).
		SetCommonHeader("Accept", "application/json")
	if httpClient != nil {
		if httpClient.Timeout != 0 {
			r.SetTimeout(httpClient.Timeout)
		}
		if httpClient.Transport != nil {
			r.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
				return httpClient.Transport.RoundTrip
			})
		}
	}

	c := &Client{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Creds:    creds,
		r:        r,
	}
	r.OnBeforeRequest(c.sign)
	return c
}

// sign attaches the four auth parameters to the request URL query.
func (c *Client) sign(_ *req.Client, r *req.Request) error {
	nonce, err := newNonce()
	if err != nil {
		return fmt.Errorf("nonce: %w", err)
	}
	ts := NowSeconds()
	tsStr := strconv.FormatInt(ts, 10)
	r.SetQueryParam("client_id", c.Creds.ClientID)
	r.SetQueryParam("nonce", nonce)
	r.SetQueryParam("ts", tsStr)
	r.SetQueryParam("sign", Sign(c.Creds.ClientKey, c.Creds.ClientID, nonce, ts))
	return nil
}

// Envelope is the standard 100x response shape.
type Envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Msg     string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
}

// APIError is returned when the server replies with a non-zero envelope code.
type APIError struct {
	Code    int
	Message string
	Status  int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("100x api: code=%d message=%q (http %d)", e.Code, e.Message, e.Status)
}

// Get sends a signed GET. `in` is converted to query parameters via req's
// struct-tag handling (`url:"name,omitempty"`).
func (c *Client) Get(ctx context.Context, path string, in, out any) error {
	r := c.r.R().SetContext(ctx)
	if in != nil {
		r.SetQueryParamsFromStruct(in)
	}
	resp, err := r.Get(path)
	if err != nil {
		return err
	}
	return decodeEnvelope(resp.Bytes(), resp.StatusCode, out)
}

// Post sends a signed POST with `in` as the JSON body.
func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	r := c.r.R().SetContext(ctx)
	if in != nil {
		r.SetBody(in)
	}
	resp, err := r.Post(path)
	if err != nil {
		return err
	}
	return decodeEnvelope(resp.Bytes(), resp.StatusCode, out)
}

func decodeEnvelope(raw []byte, status int, out any) error {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("decode envelope (http %d): %w; body=%s", status, err, truncate(raw))
	}
	msg := env.Message
	if msg == "" {
		msg = env.Msg
	}
	if env.Code != 0 {
		return &APIError{Code: env.Code, Message: msg, Status: status}
	}
	if out == nil || len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode data: %w; data=%s", err, truncate(env.Data))
	}
	return nil
}

func newNonce() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func truncate(b []byte) string {
	const maxLen = 256
	if len(b) > maxLen {
		return string(b[:maxLen]) + "..."
	}
	return string(b)
}
