// Package version exposes the build-time version metadata.
//
// The values are populated via -ldflags at link time; defaults are sane for
// `go run` and `go build` without explicit flags.
package version

// Version is the released semver string.
var Version = "dev"

// Commit is the git short SHA.
var Commit = "none"

// BuildDate is the RFC3339 build timestamp.
var BuildDate = "unknown"

// RepoSlug is the "owner/repo" GitHub identifier `100x upgrade` queries.
// Injected at build time via -ldflags; empty disables the command.
var RepoSlug = ""
