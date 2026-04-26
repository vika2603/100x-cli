// Command 100x is the entrypoint binary for the 100x futures-trading CLI.
package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/cmd/root"
	"github.com/vika2603/100x-cli/internal/exit"
	"github.com/vika2603/100x-cli/internal/prompt"
)

func main() {
	// Suppress SIGPIPE so `100x ... | head` exits cleanly: the runtime
	// would otherwise terminate the process with a "broken pipe" error
	// the first time it tried to write to a closed stdout.
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(run())
}

// run owns the deferred signal-context teardown so os.Exit never bypasses it.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd, emitErr := root.NewCmdRoot()
	err := cmd.ExecuteContext(ctx)
	if err == nil {
		return exit.OK
	}
	if errors.Is(err, syscall.EPIPE) {
		// Downstream consumer closed its stdin; nothing more to write.
		return exit.OK
	}
	code, codeString := classify(err)
	emitErr(err, code, codeString)
	return code
}

// classify maps an error to (exit code, stable string code). The string
// code is the closed set callers may branch on programmatically; the
// numeric code is the process exit status.
func classify(err error) (int, string) {
	if err == nil {
		return exit.OK, "ok"
	}
	if errors.Is(err, prompt.ErrDestructiveNoTTY) {
		return exit.NonTTY, "non_tty"
	}
	if futures.IsAuth(err) {
		return exit.Auth, "auth"
	}
	if futures.IsValidation(err) {
		return exit.Validation, "validation"
	}
	if errors.Is(err, context.Canceled) {
		// signal.NotifyContext cancels ctx on SIGINT/SIGTERM; map to the
		// conventional 130 so cron and shell wrappers see a recognisable
		// interruption rather than a generic abort.
		return exit.Interrupted, "interrupted"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return exit.Network, "network"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return exit.Network, "network"
	}
	return exit.Generic, "generic"
}
