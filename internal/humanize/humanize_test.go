package humanize

import (
	"testing"
	"time"
)

func TestBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1 << 20, "1.0 MiB"},
		{1 << 30, "1.0 GiB"},
	}
	for _, c := range cases {
		if got := Bytes(c.in); got != c.want {
			t.Errorf("Bytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCount(t *testing.T) {
	cases := map[int]string{
		0:       "0",
		7:       "7",
		999:     "999",
		1000:    "1,000",
		12345:   "12,345",
		1000000: "1,000,000",
		-12345:  "-12,345",
	}
	for in, want := range cases {
		if got := Count(in); got != want {
			t.Errorf("Count(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRelTimeFrom(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{0, "just now"},
		{30 * time.Second, "30s ago"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{2 * 24 * time.Hour, "2d ago"},
	}
	for _, c := range cases {
		if got := relTimeFrom(now, now.Add(-c.ago)); got != c.want {
			t.Errorf("relTimeFrom(-%s) = %q, want %q", c.ago, got, c.want)
		}
	}
	// Older than a week falls back to an absolute date.
	old := now.Add(-30 * 24 * time.Hour)
	if got := relTimeFrom(now, old); got != old.Format("2006-01-02 15:04") {
		t.Errorf("old RelTime = %q", got)
	}
}

func TestRatio(t *testing.T) {
	if got := Ratio(1000, 250); got != "4.0x" {
		t.Errorf("Ratio = %q, want 4.0x", got)
	}
	if got := Ratio(1, 0); got != "—" {
		t.Errorf("Ratio div0 = %q, want —", got)
	}
}
