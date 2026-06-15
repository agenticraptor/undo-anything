package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agenticraptor/undo-anything/internal/config"
	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// dirArg returns the first positional argument as a directory, or "." .
func dirArg(args []string) string {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return args[0]
	}
	return "."
}

// openStore finds the store for the given directory, searching upward like git
// so commands work from any subdirectory.
func openStore(dir string) (*store.Store, error) {
	return store.Find(dir)
}

// loadConfigured opens the store for dir and loads its config in one step.
func loadConfigured(dir string) (*store.Store, config.Config, error) {
	s, err := openStore(dir)
	if err != nil {
		return nil, config.Config{}, err
	}
	cfg, err := config.Load(s.Dir)
	if err != nil {
		return nil, config.Config{}, err
	}
	return s, cfg, nil
}

// buildMatcher assembles the ignore matcher from defaults, config patterns, an
// optional .uaignore file, and (optionally) the folder's .gitignore.
func buildMatcher(s *store.Store, cfg config.Config) (*ignore.Matcher, error) {
	extra := append([]string{}, cfg.Ignore.Patterns...)

	if uaPatterns, err := ignore.LoadFile(filepath.Join(s.Root, ".uaignore")); err != nil {
		return nil, err
	} else {
		extra = append(extra, uaPatterns...)
	}

	if cfg.Ignore.UseGitignore {
		if gitPatterns, err := ignore.LoadFile(filepath.Join(s.Root, ".gitignore")); err != nil {
			return nil, err
		} else {
			extra = append(extra, gitPatterns...)
		}
	}
	return ignore.NewWithDefaults(extra), nil
}

// confirm prompts the user for a yes/no answer unless assumeYes is set.
func confirm(prompt string, assumeYes bool) bool {
	if assumeYes {
		return true
	}
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// toRepoRel converts a user-supplied path (relative to the current directory,
// absolute, or prefixed with ./) into the root-relative, slash-separated form
// snapshots are keyed by. This lets `ua restore <id> main.go` work from any
// subdirectory. If the path can't be made relative to the root it is returned
// in slash form unchanged, so the caller surfaces a clear "not in snapshot"
// error rather than silently doing the wrong thing.
func toRepoRel(root, p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// relForDisplay shortens an absolute path to be relative to the user's home
// directory for friendlier output.
func relForDisplay(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
