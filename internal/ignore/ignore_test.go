package ignore

import "testing"

func TestMatch(t *testing.T) {
	m := New([]string{
		"node_modules/",
		"*.log",
		"/build",
		"docs/*.tmp",
		"**/cache/**",
		"!keep.log",
	})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"node_modules", true, true},
		{"node_modules/react/index.js", false, true}, // nested under matched dir
		{"app.log", false, true},
		{"logs/app.log", false, true}, // unanchored basename match at any depth
		{"keep.log", false, false},    // negation wins (last match)
		{"build", true, true},         // anchored to root
		{"src/build", true, false},    // anchored: only root-level build
		{"docs/notes.tmp", false, true},
		{"docs/sub/notes.tmp", false, false}, // docs/*.tmp is one level only
		{"a/cache/b/file", false, true},      // ** on both sides
		{"src/main.go", false, false},
		{"README.md", false, false},
	}
	for _, c := range cases {
		if got := m.Match(c.path, c.isDir); got != c.want {
			t.Errorf("Match(%q, dir=%v) = %v, want %v", c.path, c.isDir, got, c.want)
		}
	}
}

func TestDefaultsIgnoreStore(t *testing.T) {
	m := NewWithDefaults(nil)
	if !m.Match(".undo", true) {
		t.Error(".undo should be ignored by default")
	}
	if !m.Match(".git/config", false) {
		t.Error(".git contents should be ignored by default")
	}
	if !m.Match("project/.DS_Store", false) {
		t.Error(".DS_Store should be ignored at any depth")
	}
}

func TestCommentsAndBlankLines(t *testing.T) {
	m := New([]string{"", "  ", "# a comment", "secret.txt"})
	if len(m.patterns) != 1 {
		t.Fatalf("expected 1 effective pattern, got %d", len(m.patterns))
	}
	if !m.Match("secret.txt", false) {
		t.Error("secret.txt should match")
	}
}

func TestNegationOrder(t *testing.T) {
	// A later negation re-includes a path excluded earlier.
	m := New([]string{"*.env", "!.env.example"})
	if !m.Match("prod.env", false) {
		t.Error("prod.env should be ignored")
	}
	if m.Match(".env.example", false) {
		t.Error(".env.example should be re-included by negation")
	}
}
