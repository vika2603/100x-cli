package exit

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/prompt"
)

// Classify maps an error to (exit code, stable string code). The string code
// is the closed set callers may branch on programmatically; the numeric code
// is the process exit status.
func Classify(err error) (int, string) {
	if err == nil {
		return OK, "ok"
	}
	var coded *CodedError
	if errors.As(err, &coded) {
		return coded.Code, coded.Stable
	}
	if isUsageError(err) {
		return Usage, "usage"
	}
	if errors.Is(err, prompt.ErrDestructiveNoTTY) {
		return Aborted, "cancelled"
	}
	if futures.IsAuth(err) {
		return Auth, "auth"
	}
	if futures.IsRateLimited(err) {
		return RateLimited, "rate_limited"
	}
	if futures.IsServer(err) {
		return Network, "server"
	}
	if futures.IsBusiness(err) {
		return Business, "business"
	}
	if errors.Is(err, context.Canceled) {
		return Aborted, "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Network, "network"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return Network, "network"
	}
	return Generic, "generic"
}

func isUsageError(err error) bool {
	msg := err.Error()
	if msg == "" {
		return false
	}
	prefixes := []string{
		"unknown command ",
		"unknown flag: ",
		"required flag(s) ",
		"accepts ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
