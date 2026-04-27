// Command 100x is the entrypoint binary for the 100x futures-trading CLI.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/vika2603/100x-cli/internal/cmd/root"
	"github.com/vika2603/100x-cli/internal/exit"
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
	code, codeString := exit.Classify(err)
	emitErr(err, code, codeString)
	return code
}
