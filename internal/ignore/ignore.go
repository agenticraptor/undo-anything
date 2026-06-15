// Package ignore implements gitignore-style path matching used to decide which
// files undo-anything tracks. It supports the common subset of .gitignore
// syntax: comments, blank lines, negation (!), directory-only patterns (a
// trailing slash), anchored patterns (a leading or embedded slash), and the
// `*`, `?`, and `**` wildcards.
package ignore

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// DefaultPatterns are always ignored. They keep the store small and avoid
// snapshotting noisy build output, VCS metadata, and the store itself.
var DefaultPatterns = []string{
	".undo/",
	".git/",
	".hg/",
	".svn/",
	"node_modules/",
	".venv/",
	"venv/",
	"__pycache__/",
	"target/",
	"dist/",
	"build/",
	".next/",
	".cache/",
	".DS_Store",
	"*.log",
	"*.tmp",
	"*.swp",
	"*.swx",
	"*~",
}

type pattern struct {
	raw      string
	negate   bool
	dirOnly  bool
	anchored bool
	segs     []string // pattern split on '/'
}

// Matcher tests relative, slash-separated paths against an ordered set of
// patterns. The last matching pattern wins, mirroring git's semantics.
type Matcher struct {
	patterns []pattern
}

// New builds a Matcher from raw gitignore-style lines.
func New(lines []string) *Matcher {
	m := &Matcher{}
	for _, line := range lines {
		if p, ok := parse(line); ok {
			m.patterns = append(m.patterns, p)
		}
	}
	return m
}

// NewWithDefaults builds a Matcher seeded with DefaultPatterns followed by the
// supplied user patterns (so user rules can override the defaults).
func NewWithDefaults(extra []string) *Matcher {
	return New(append(append([]string{}, DefaultPatterns...), extra...))
}

// LoadFile reads gitignore-style patterns from a file. A missing file is not an
// error and yields an empty slice.
func LoadFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return readLines(f)
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func parse(line string) (pattern, bool) {
	raw := line
	// Trim trailing whitespace that is not escaped.
	trimmed := strings.TrimRight(line, " \t")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return pattern{}, false
	}
	p := pattern{raw: raw}
	if strings.HasPrefix(trimmed, "!") {
		p.negate = true
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "/") {
		p.dirOnly = true
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	// A leading slash, or a slash anywhere but the very end, anchors the
	// pattern to the matcher root.
	if strings.HasPrefix(trimmed, "/") {
		p.anchored = true
		trimmed = strings.TrimPrefix(trimmed, "/")
	} else if strings.Contains(trimmed, "/") {
		p.anchored = true
	}
	if trimmed == "" {
		return pattern{}, false
	}
	p.segs = strings.Split(trimmed, "/")
	return p, true
}

// Match reports whether the given relative path (slash-separated) is ignored.
// isDir indicates whether the path refers to a directory. The last matching
// pattern wins, mirroring git.
func (m *Matcher) Match(path string, isDir bool) bool {
	path = strings.TrimPrefix(path, "./")
	path = strings.Trim(path, "/")
	if path == "" {
		return false
	}
	ignored := false
	for _, p := range m.patterns {
		if p.matches(path, isDir) {
			ignored = !p.negate
		}
	}
	return ignored
}

// matches reports whether pattern p matches the given path. A directory-only
// pattern (trailing slash) matches the directory itself only when isDir is
// true, but always matches files nested *beneath* a matched directory — which
// is how `node_modules/` ignores `node_modules/react/index.js`.
func (p pattern) matches(path string, isDir bool) bool {
	parts := strings.Split(path, "/")
	var anyExact, anyAncestor bool

	consider := func(matched, exact bool) {
		if !matched {
			return
		}
		if exact {
			anyExact = true
		} else {
			anyAncestor = true
		}
	}

	if p.anchored {
		consider(matchSegs(p.segs, parts))
	} else {
		// Unanchored: the pattern may match starting at any path segment.
		for i := range parts {
			consider(matchSegs(p.segs, parts[i:]))
		}
	}

	if !anyExact && !anyAncestor {
		return false
	}
	if !p.dirOnly {
		return true
	}
	// Dir-only: match a nested file (ancestor match) always, but a direct
	// match only when the path is itself a directory.
	return anyAncestor || (anyExact && isDir)
}

// matchSegs matches a slash-split pattern against slash-split path segments,
// honoring `**` (which matches zero or more segments). It returns whether the
// pattern matched and, if so, whether it consumed the path exactly through its
// final segment (exact) versus matching an ancestor with deeper segments
// remaining (not exact).
func matchSegs(pat, name []string) (matched, exact bool) {
	if len(pat) == 0 {
		return true, len(name) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(name); i++ {
			if m, e := matchSegs(pat[1:], name[i:]); m {
				return true, e
			}
		}
		return false, false
	}
	if len(name) == 0 {
		return false, false
	}
	if !matchSegment(pat[0], name[0]) {
		return false, false
	}
	if len(pat) == 1 {
		// The final pattern segment matches this path segment (and everything
		// nested beneath it). It's an exact match only if no deeper segments
		// remain.
		return true, len(name) == 1
	}
	return matchSegs(pat[1:], name[1:])
}

// matchSegment performs shell-style matching of a single segment with support
// for `*` and `?`. `*` does not cross a path separator (already split out).
func matchSegment(pat, name string) bool {
	pi, ni := 0, 0
	star := -1
	starName := 0
	for ni < len(name) {
		if pi < len(pat) && (pat[pi] == name[ni] || pat[pi] == '?') {
			pi++
			ni++
		} else if pi < len(pat) && pat[pi] == '*' {
			star = pi
			starName = ni
			pi++
		} else if star != -1 {
			pi = star + 1
			starName++
			ni = starName
		} else {
			return false
		}
	}
	for pi < len(pat) && pat[pi] == '*' {
		pi++
	}
	return pi == len(pat)
}
