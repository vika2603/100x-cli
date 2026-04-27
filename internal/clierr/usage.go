// Package clierr classifies command-line input errors for stable exits.
package clierr

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vika2603/100x-cli/internal/exit"
)

// Usage wraps command-line input mistakes with the stable usage exit code.
func Usage(err error) error {
	if err == nil {
		return nil
	}
	return exit.NewCodedError(exit.Usage, "usage", err)
}

// IsUsage reports whether err is already classified as a usage mistake.
func IsUsage(err error) bool {
	var coded *exit.CodedError
	return errors.As(err, &coded) && coded.Stable == "usage"
}

// Usagef formats a command-line input mistake.
func Usagef(format string, args ...any) error {
	return Usage(fmt.Errorf(format, args...))
}

// WithHelpHint appends a short usage hint to human and JSON error messages.
func WithHelpHint(err error, commandPath string) error {
	if err == nil || commandPath == "" {
		return err
	}
	if strings.Contains(err.Error(), "--help") {
		return err
	}
	return fmt.Errorf("%w. Run `%s --help` for usage", err, commandPath)
}

// PositiveInt validates pagination and limit flags.
func PositiveInt(name string, value int) error {
	if value <= 0 {
		return Usagef("%s must be greater than 0", name)
	}
	return nil
}

// PositiveID validates positional numeric ids before a request reaches the API.
func PositiveID(name, value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return Usagef("%s must be a positive integer", name)
	}
	return nil
}

// PositiveNumber validates decimal-valued string inputs without changing their
// wire representation.
func PositiveNumber(name, value string) error {
	if value == "" {
		return nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 {
		return Usagef("%s must be a positive number", name)
	}
	return nil
}
