package futures

import (
	"errors"

	"github.com/vika2603/100x-cli/api/internal/transport"
)

// APIError is the typed error returned for non-zero envelope codes.
// Re-exported so consumers do not need to import the transport package.
type APIError = transport.APIError

// IsAuth reports whether err is one of the documented credential failures.
func IsAuth(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.Code {
	case 10003, 10004, 10009, 10024, 10025:
		return true
	default:
		return false
	}
}

// IsRateLimited reports whether err is a retryable rate-limit/capacity error.
func IsRateLimited(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.Code {
	case 10006, 10015:
		return true
	default:
		return false
	}
}

// IsServer reports whether err is a backend/engine failure.
func IsServer(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.Code {
	case 10001, 10018, 10019:
		return true
	default:
		return false
	}
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
