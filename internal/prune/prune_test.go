package prune

import (
	"testing"
	"time"

	"github.com/agenticraptor/undo-anything/internal/store"
)

func TestPlanRetention(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	mk := func(id string, ago time.Duration) store.IndexEntry {
		return store.IndexEntry{ID: id, Time: now.Add(-ago)}
	}
	// Chronological (oldest first).
	entries := []store.IndexEntry{
		mk("old10d", 10*24*time.Hour),           // beyond daily window -> drop
		mk("day5a", 5*24*time.Hour+3*time.Hour), // same day as day5b, older -> drop
		mk("day5b", 5*24*time.Hour),             // newest of that day -> keep
		mk("day2", 2*24*time.Hour),              // within daily window -> keep
		mk("recent1h", 1*time.Hour),             // within keep-hours -> keep
		mk("newest", 5*time.Minute),             // newest -> keep
	}
	policy := Policy{KeepHours: 24, KeepDaily: 7, MaxSnapshots: 1000}

	keep, remove := Plan(entries, policy, now)

	mustKeep := []string{"newest", "recent1h", "day2", "day5b"}
	for _, id := range mustKeep {
		if !keep[id] {
			t.Errorf("expected to keep %s", id)
		}
	}
	mustDrop := []string{"old10d", "day5a"}
	for _, id := range mustDrop {
		if keep[id] {
			t.Errorf("expected to drop %s", id)
		}
	}
	if len(remove) != 2 {
		t.Errorf("remove count = %d, want 2 (%v)", len(remove), remove)
	}
}

func TestPlanAlwaysKeepsNewest(t *testing.T) {
	now := time.Now()
	entries := []store.IndexEntry{
		{ID: "a", Time: now.Add(-100 * 24 * time.Hour)},
		{ID: "b", Time: now.Add(-99 * 24 * time.Hour)},
	}
	// Aggressive policy that would drop everything old.
	keep, _ := Plan(entries, Policy{KeepHours: 1, KeepDaily: 1, MaxSnapshots: 1000}, now)
	if !keep["b"] {
		t.Error("newest snapshot must always be kept")
	}
}

func TestPlanMaxSnapshotsCap(t *testing.T) {
	now := time.Now()
	var entries []store.IndexEntry
	// Build chronological (oldest first): entry i is (9-i) minutes old, so the
	// last entries ("h","i","j") are the most recent.
	for i := 0; i < 10; i++ {
		entries = append(entries, store.IndexEntry{
			ID:   string(rune('a' + i)),
			Time: now.Add(-time.Duration(9-i) * time.Minute), // all within keep window
		})
	}
	keep, remove := Plan(entries, Policy{KeepHours: 24, KeepDaily: 7, MaxSnapshots: 3}, now)
	if len(keep) != 3 {
		t.Errorf("cap not enforced: kept %d, want 3", len(keep))
	}
	if len(remove) != 7 {
		t.Errorf("remove = %d, want 7", len(remove))
	}
	// The three most recent ("h","i","j") should be the survivors.
	for _, id := range []string{"h", "i", "j"} {
		if !keep[id] {
			t.Errorf("expected most-recent %s kept under cap", id)
		}
	}
}

func TestPlanEmpty(t *testing.T) {
	keep, remove := Plan(nil, Policy{KeepHours: 24}, time.Now())
	if len(keep) != 0 || len(remove) != 0 {
		t.Error("empty input should yield empty plan")
	}
}
