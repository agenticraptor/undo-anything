// Package buildinfo exposes version metadata stamped in at build time via
// -ldflags. The zero values below are used for `go run` / `go install` builds
// where no ldflags are supplied.
package buildinfo

import "fmt"

// These variables are overridden at build time with -X linker flags. See the
// Makefile and .goreleaser.yaml.
var (
	// Version is the semantic version (e.g. "v0.1.0") or "dev".
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// Date is the RFC3339 build timestamp.
	Date = "unknown"
)

// String returns a human-readable one-line version summary.
func String() string {
	return fmt.Sprintf("undo-anything %s (commit %s, built %s)", Version, Commit, Date)
}
