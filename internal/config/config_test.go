package config

import (
	"testing"
)

func TestDefaults(t *testing.T) {
	c := Default()
	if c.Watch.DebounceMs != 1500 {
		t.Errorf("DebounceMs = %d, want 1500", c.Watch.DebounceMs)
	}
	if !c.Ignore.UseGitignore {
		t.Error("UseGitignore should default true")
	}
	if c.MaxFileSizeBytes() != 50<<20 {
		t.Errorf("MaxFileSizeBytes = %d, want %d", c.MaxFileSizeBytes(), 50<<20)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if c.Retention.KeepHours != Default().Retention.KeepHours {
		t.Error("missing config should fall back to defaults")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := Default()
	in.Watch.DebounceMs = 750
	in.Ignore.Patterns = []string{"*.bin", "tmp/"}
	in.Retention.MaxSnapshots = 42

	if err := Save(dir, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Watch.DebounceMs != 750 {
		t.Errorf("DebounceMs = %d, want 750", out.Watch.DebounceMs)
	}
	if len(out.Ignore.Patterns) != 2 || out.Ignore.Patterns[0] != "*.bin" {
		t.Errorf("Patterns = %v", out.Ignore.Patterns)
	}
	if out.Retention.MaxSnapshots != 42 {
		t.Errorf("MaxSnapshots = %d, want 42", out.Retention.MaxSnapshots)
	}
}

func TestMaxFileSizeZeroWhenUnset(t *testing.T) {
	var c Config
	if c.MaxFileSizeBytes() != 0 {
		t.Errorf("unset MaxFileSizeMB should yield 0, got %d", c.MaxFileSizeBytes())
	}
}
