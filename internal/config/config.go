// Package config loads and saves a store's TOML configuration, which lives at
// .undo/config.toml. Every field has a sensible zero-config default, so the
// file is optional.
package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// FileName is the per-store config filename inside the .undo directory.
const FileName = "config.toml"

// Config holds all user-tunable behavior for a single watched folder.
type Config struct {
	Watch     WatchConfig     `toml:"watch"`
	Ignore    IgnoreConfig    `toml:"ignore"`
	Retention RetentionConfig `toml:"retention"`
}

// WatchConfig tunes the file-watching daemon.
type WatchConfig struct {
	// DebounceMs coalesces rapid successive saves into one snapshot.
	DebounceMs int `toml:"debounce_ms"`
	// MaxFileSizeMB skips files larger than this when snapshotting.
	MaxFileSizeMB int64 `toml:"max_file_size_mb"`
}

// IgnoreConfig controls which paths are excluded.
type IgnoreConfig struct {
	// Patterns are extra .gitignore-style rules layered on the built-in defaults.
	Patterns []string `toml:"patterns"`
	// UseGitignore also honors the folder's .gitignore when true.
	UseGitignore bool `toml:"use_gitignore"`
}

// RetentionConfig describes how `ua prune` thins history.
type RetentionConfig struct {
	// KeepHours retains every snapshot newer than this many hours.
	KeepHours int `toml:"keep_hours"`
	// KeepDaily retains one snapshot per day for this many recent days
	// (applied to snapshots older than KeepHours).
	KeepDaily int `toml:"keep_daily"`
	// MaxSnapshots is a hard cap; the newest are always kept.
	MaxSnapshots int `toml:"max_snapshots"`
}

// Default returns the zero-config defaults used when no file is present.
func Default() Config {
	return Config{
		Watch: WatchConfig{
			DebounceMs:    1500,
			MaxFileSizeMB: 50,
		},
		Ignore: IgnoreConfig{
			Patterns:     []string{},
			UseGitignore: true,
		},
		Retention: RetentionConfig{
			KeepHours:    48,
			KeepDaily:    30,
			MaxSnapshots: 2000,
		},
	}
}

// Path returns the config path for a given .undo directory.
func Path(storeDir string) string {
	return filepath.Join(storeDir, FileName)
}

// Load reads config from a store directory, returning defaults if no file
// exists. Unknown keys are ignored so configs are forward-compatible.
func Load(storeDir string) (Config, error) {
	cfg := Default()
	path := Path(storeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes config to a store directory as documented TOML.
func Save(storeDir string, cfg Config) error {
	var buf bytes.Buffer
	buf.WriteString(header)
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(Path(storeDir), buf.Bytes(), 0o644)
}

// MaxFileSizeBytes converts the configured MB cap to bytes.
func (c Config) MaxFileSizeBytes() int64 {
	if c.Watch.MaxFileSizeMB <= 0 {
		return 0
	}
	return c.Watch.MaxFileSizeMB << 20
}

const header = `# undo-anything configuration
# Docs: https://github.com/agenticraptor/undo-anything/blob/main/docs/configuration.md
# Every value below is optional and falls back to a sensible default.

`
