// Package exit centralises the CLI's exit-code contract.
//
// All command code returns typed errors; cmd/100x/main.go inspects the error
// and returns the matching code. Adding a new failure class means adding a
// constant here and mapping it from main.
package exit

// CodedError carries an explicit CLI exit-code classification across package
// boundaries while preserving the wrapped cause for error messages.
type CodedError struct {
	Code   int
	Stable string
	Err    error
}

// NewCodedError wraps err with a numeric exit code and stable string code.
func NewCodedError(code int, stable string, err error) *CodedError {
	return &CodedError{Code: code, Stable: stable, Err: err}
}

func (e *CodedError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CodedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

const (
	// OK is reserved for completed work with no error.
	OK = 0

	// Generic is the default non-success code for unclassified errors.
	Generic = 1

	// Usage covers flag-parse failures, unknown commands, missing required args.
	Usage = 2

	// Auth covers signature, credential, and permission failures (backend 10xxx).
	Auth = 3

	// RateLimited covers rate-limit and temporary capacity failures.
	RateLimited = 4

	// Business covers backend rejections of well-formed requests: validation,
	// missing balance, closed markets, missing orders, and similar cases.
	Business = 5

	// Network covers transport, DNS, and connection failures.
	Network = 6

	// Aborted covers user-initiated aborts (Ctrl+C, declined confirmations).
	Aborted = 7
)
