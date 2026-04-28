// Package version exposes the build-time version metadata as a single
// struct so callers don't reach for individual package vars and don't have
// to repeat the goreleaser/git-describe normalization.
//
// The four lowercase package vars below are the only symbols the linker
// writes to via -ldflags -X. The exported Info instance Current is
// composed from them at init time and is the one read path the rest of
// the code uses.
package version

import "strings"

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
	repoSlug  = ""
)

// Info bundles the build-time identity of a 100x binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	RepoSlug  string
}

// Current is the Info for this binary, populated from the linker-injected
// package vars after -ldflags -X has overwritten their static initializers.
var Current = Info{
	Version:   normalizeVersion(version),
	Commit:    commit,
	BuildDate: buildDate,
	RepoSlug:  repoSlug,
}

// normalizeVersion gives every release tag the leading "v" prefix.
// Goreleaser strips it from {{.Version}}; `git describe` keeps it. We
// always want the displayed and compared form to be "vX.Y.Z".
func normalizeVersion(v string) string {
	if v == "" || v == "dev" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}
