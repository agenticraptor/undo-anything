// Package humanize formats sizes, counts, and durations for friendly CLI output.
package humanize

import (
	"fmt"
	"math"
	"time"
)

// Bytes formats a byte count using binary units (KiB, MiB, ...).
func Bytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}

// Count formats an integer with thousands separators (e.g. 12,345).
func Count(n int) string {
	if n < 0 {
		return "-" + Count(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	pre := len(s) % 3
	if pre > 0 {
		out = append(out, s[:pre]...)
	}
	for i := pre; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

// RelTime renders a time as a compact "time ago" string, falling back to an
// absolute timestamp for anything older than a week.
func RelTime(t time.Time) string {
	return relTimeFrom(time.Now(), t)
}

func relTimeFrom(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 0:
		return "in the future"
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(math.Floor(d.Hours()/24)))
	default:
		return t.Format("2006-01-02 15:04")
	}
}

// Ratio renders a dedup/compression ratio like "4.2x".
func Ratio(numerator, denominator int64) string {
	if denominator == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1fx", float64(numerator)/float64(denominator))
}
