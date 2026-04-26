// Package exit centralises the CLI's exit-code contract.
//
// All command code returns typed errors; cmd/100x/main.go inspects the error
// and returns the matching code. Adding a new failure class means adding a
// constant here and mapping it from main.
package exit

const (
	// OK is reserved for completed work with no error.
	OK = 0

	// Generic is the default non-success code for unclassified errors.
	Generic = 1

	// Usage covers flag-parse failures, unknown commands, missing required args.
	Usage = 2

	// Auth covers signature, credential, and permission failures (backend 10xxx).
	Auth = 3

	// Validation covers backend rejections of well-formed but invalid requests
	// (backend 20xxx).
	Validation = 4

	// NotFound covers operations against ids that do not exist.
	NotFound = 5

	// Network covers transport, DNS, and connection failures.
	Network = 6

	// Aborted covers user-initiated aborts (Ctrl+C, declined confirmations).
	Aborted = 7

	// NonTTY signals that an interactive prompt was needed (a destructive
	// op without -y) but no terminal was attached.
	NonTTY = 73

	// Interrupted is the conventional exit code for SIGINT propagation.
	Interrupted = 130
)
