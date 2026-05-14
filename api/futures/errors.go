package futures

import (
	"errors"
	"slices"

	"github.com/vika2603/100x-cli/api/internal/transport"
)

// APIError is the typed error returned for non-zero envelope codes.
// Re-exported so consumers do not need to import the transport package.
type APIError = transport.APIError

// Gateway envelope codes used to classify APIError values into the
// auth / rate-limit / server / business categories. Update these slices
// when the gateway adds or renumbers codes; the IsXxx helpers below pick
// them up automatically.
var (
	// authCodes: signature, nonce, client_id, permission, key state failures.
	authCodes = []int{10003, 10004, 10009, 10024, 10025}

	// rateLimitCodes: per-account or per-IP throttling, queue-full.
	rateLimitCodes = []int{10006, 10015}

	// serverCodes: matching-engine / backend internal failures.
	serverCodes = []int{10001, 10018, 10019}
)

// IsAuth reports whether err is one of the documented credential failures.
func IsAuth(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return slices.Contains(authCodes, ae.Code)
}

// IsRateLimited reports whether err is a retryable rate-limit/capacity error.
func IsRateLimited(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return slices.Contains(rateLimitCodes, ae.Code)
}

// IsServer reports whether err is a backend/engine failure.
func IsServer(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return slices.Contains(serverCodes, ae.Code)
}

// IsBusiness reports whether err is a backend rejection of a well-formed
// request, including 1xxxx business codes and 2xxxx parameter validation codes.
func IsBusiness(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	if IsAuth(err) || IsRateLimited(err) || IsServer(err) {
		return false
	}
	return (ae.Code >= 10000 && ae.Code < 20000) || (ae.Code >= 20000 && ae.Code < 30000)
}
