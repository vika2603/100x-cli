// Package output centralises the CLI's stdout / stderr contract.
//
// stdout carries data only; stderr carries logs, prompts, and errors.
// Commands must use writers from this package rather than os.Stdout /
// os.Stderr so that the three-stream guarantee holds across the surface.
//
// Format selection (FormatHuman vs FormatJSON), gojq filtering, and quiet
// mode are root-flag controlled and applied uniformly through Render.
package output

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// Format selects how Render serialises a payload.
type Format int

// Format values.
const (
	FormatHuman Format = iota
	FormatJSON
)

// ColorMode selects whether ANSI escape sequences are emitted.
type ColorMode int

// ColorMode values map to --color flag values. ColorAuto follows
// per-stream tty detection and the NO_COLOR env var; the other two
// override regardless of context.
const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

// Renderer holds the per-invocation output state populated by root cmd flags.
type Renderer struct {
	Out    io.Writer // stdout
	Err    io.Writer // stderr
	Format Format
	JQ     string    // optional gojq expression applied to JSON payloads
	Quiet  bool      // when true, only ids / minimal markers go to stdout
	Color  ColorMode // controls ANSI emission; auto-detects per stream
}

// New returns a Renderer wired to the real OS file handles.
func New() *Renderer {
	return &Renderer{Out: os.Stdout, Err: os.Stderr, Format: FormatHuman}
}

// ColorOnStdout reports whether ANSI escapes should be emitted on stdout
// for the current invocation. It follows the precedence:
//
//	--color never  → false
//	--color always → true
//	NO_COLOR set   → false
//	otherwise      → true iff stdout is a tty
func (r *Renderer) ColorOnStdout() bool {
	return r.colorFor(r.Out)
}

// ColorOnStderr is the stderr equivalent of ColorOnStdout.
func (r *Renderer) ColorOnStderr() bool {
	return r.colorFor(r.Err)
}

func (r *Renderer) colorFor(w io.Writer) bool {
	switch r.Color {
	case ColorNever:
		return false
	case ColorAlways:
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

// Println writes one line of human-meant text to stderr.
//
// Used for progress / status messages: stdout stays clean for piped consumers.
func (r *Renderer) Println(args ...any) {
	if r.Quiet {
		return
	}
	_, _ = fmt.Fprintln(r.Err, args...)
}

// Resultln writes one line of command result data to stdout.
func (r *Renderer) Resultln(args ...any) error {
	if r.Quiet {
		return nil
	}
	_, err := fmt.Fprintln(r.Out, args...)
	return err
}

// Emptyln writes a human-readable empty-state message to stdout.
func (r *Renderer) Emptyln(message string) error {
	if r.Quiet {
		return nil
	}
	_, err := fmt.Fprintln(r.Out, message)
	return err
}
