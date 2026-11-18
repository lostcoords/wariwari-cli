package commands

import (
	"testing"
	"time"
)

func TestExpiresInWindows(t *testing.T) {
	invitationNow = func() time.Time { return time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { invitationNow = time.Now })

	for window, want := range map[string]string{
		"24h": "2026-12-02T09:00:00Z",
		"7d":  "2026-12-08T09:00:00Z",
		"30d": "2026-12-31T09:00:00Z",
		"90m": "2026-12-01T10:30:00Z",
	} {
		got, err := expiresAt(window)
		if err != nil || got != want {
			t.Fatalf("expiresAt(%q) = %q, %v; want %q", window, got, err, want)
		}
	}
	for _, window := range []string{"", "soon", "0d", "-1h", "1w"} {
		if _, err := expiresAt(window); err == nil {
			t.Fatalf("expiresAt(%q) accepted a window it should refuse", window)
		}
	}
}
