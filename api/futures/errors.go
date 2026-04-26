package futures

import (
	"errors"

	"github.com/vika2603/100x-cli/api/internal/transport"
)

// APIError is the typed error returned for non-zero envelope codes.
// Re-exported so consumers do not need to import the transport package.
type APIError = transport.APIError

// IsAuth reports whether err originates from a system or signature failure.
// Backend codes 10xxx are reserved for system / authentication failures.
func IsAuth(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return ae.Code >= 10000 && ae.Code < 20000
}

// IsValidation reports whether err originates from a request validation failure.
// Backend codes 20xxx are reserved for validation.
func IsValidation(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return ae.Code >= 20000 && ae.Code < 30000
}
