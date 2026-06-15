// Package prune translates a retention policy into the concrete set of
// snapshots to keep, then drives the store's garbage collector. The policy is
// deliberately conservative: it always keeps the most recent snapshot and
// everything within the recent "keep everything" window.
package prune

import (
	"time"

	"github.com/agenticraptor/undo-anything/internal/config"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// Policy is the subset of config needed to compute a keep-set.
type Policy struct {
	KeepHours    int
	KeepDaily    int
	MaxSnapshots int
}

// FromConfig builds a Policy from a RetentionConfig.
func FromConfig(r config.RetentionConfig) Policy {
	return Policy{KeepHours: r.KeepHours, KeepDaily: r.KeepDaily, MaxSnapshots: r.MaxSnapshots}
}

// Plan returns the set of snapshot IDs to keep and the list of timeline entries
// that would be removed, without modifying anything. `now` is injectable for
// testing.
func Plan(entries []store.IndexEntry, p Policy, now time.Time) (keep map[string]bool, remove []store.IndexEntry) {
	keep = map[string]bool{}
	if len(entries) == 0 {
		return keep, nil
	}

	// entries are chronological (oldest first); walk newest first.
	ordered := make([]store.IndexEntry, len(entries))
	copy(ordered, entries)
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}

	// Always keep the most recent snapshot.
	keep[ordered[0].ID] = true

	recentCutoff := now.Add(-time.Duration(maxInt(p.KeepHours, 0)) * time.Hour)
	dailyCutoff := now.AddDate(0, 0, -maxInt(p.KeepDaily, 0))
	seenDay := map[string]bool{}

	for _, e := range ordered {
		switch {
		case !e.Time.Before(recentCutoff):
			// Within the keep-everything window.
			keep[e.ID] = true
		case p.KeepDaily > 0 && !e.Time.Before(dailyCutoff):
			// Older than the window but within the daily-retention range:
			// keep the newest snapshot per calendar day.
			day := e.Time.Format("2006-01-02")
			if !seenDay[day] {
				seenDay[day] = true
				keep[e.ID] = true
			}
		}
	}

	// Enforce the hard cap: keep at most MaxSnapshots, preferring the newest.
	if p.MaxSnapshots > 0 {
		count := 0
		capped := map[string]bool{}
		for _, e := range ordered {
			if keep[e.ID] && !capped[e.ID] {
				if count >= p.MaxSnapshots {
					continue
				}
				capped[e.ID] = true
				count++
			}
		}
		keep = capped
		keep[ordered[0].ID] = true // never drop the newest
	}

	// Build the remove list (unique IDs not kept).
	removedID := map[string]bool{}
	for _, e := range ordered {
		if !keep[e.ID] && !removedID[e.ID] {
			removedID[e.ID] = true
			remove = append(remove, e)
		}
	}
	return keep, remove
}

// Run computes the keep-set for the store and garbage-collects everything else.
func Run(s *store.Store, p Policy, now time.Time) (store.GCResult, int, error) {
	entries, err := s.Index()
	if err != nil {
		return store.GCResult{}, 0, err
	}
	keep, remove := Plan(entries, p, now)
	res, err := s.GC(keep)
	return res, len(remove), err
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
