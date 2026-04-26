// Package transport is the signed HTTP layer for api/futures (and any future
// product packages). It is internal to api/ so external Go consumers cannot
// import it directly; they go through futures.Client / futures.NewWithDoer.
package transport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// Sign computes the HMAC-SHA256 signature expected by the 100x open API.
// The template is fixed: client_id={cid}&nonce={nonce}&ts={ts}.
// The signature is hex-encoded lowercase.
func Sign(clientKey, clientID, nonce string, ts int64) string {
	template := "client_id=" + clientID + "&nonce=" + nonce + "&ts=" + strconv.FormatInt(ts, 10)
	mac := hmac.New(sha256.New, []byte(clientKey))
	mac.Write([]byte(template))
	return hex.EncodeToString(mac.Sum(nil))
}

// NowSeconds returns the current Unix timestamp in seconds.
// The server tolerates ±10s skew.
func NowSeconds() int64 {
	return time.Now().Unix()
}
