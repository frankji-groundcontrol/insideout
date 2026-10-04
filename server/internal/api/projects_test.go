package api

import (
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// FEAT-01: the board's headline promise — quiet projects get flagged. The
// rule: an active or planning project whose latest update (or creation,
// when it never got one) is older than the 14-day threshold is flagged;
// done/paused/archived projects are never flagged, whatever their age.
func TestProjectQuietFlagged(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-2 * 24 * time.Hour)
	stale := now.Add(-15 * 24 * time.Hour)
	k := "progress"
	c := "shipped the API"

	cases := []struct {
		name    string
		status  string
		created time.Duration // age of the project itself
		latest  *time.Time
		kind    *string
		content *string
		flagged bool
	}{
		{"active with fresh update is not flagged", "active", 20 * 24 * time.Hour, &fresh, &k, &c, false},
		{"active with stale update is flagged", "active", 20 * 24 * time.Hour, &stale, &k, &c, true},
		{"active recently created with no updates is not flagged", "active", 2 * 24 * time.Hour, nil, nil, nil, false},
		{"planning with stale creation is flagged", "planning", 16 * 24 * time.Hour, nil, nil, nil, true},
		{"paused quiet is expected, not flagged", "paused", 20 * 24 * time.Hour, &stale, &k, &c, false},
		{"done is never flagged", "done", 20 * 24 * time.Hour, &stale, &k, &c, false},
		{"archived is never flagged", "archived", 20 * 24 * time.Hour, &stale, &k, &c, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := store.ProjectWithLatest{Project: store.Project{
				Status:    tc.status,
				CreatedAt: now.Add(-tc.created),
			}, LatestUpdateKind: tc.kind, LatestUpdateContent: tc.content, LatestUpdateAt: tc.latest}
			if got := projectQuietFlagged(p, now); got != tc.flagged {
				t.Fatalf("projectQuietFlagged(%s) = %v, want %v", tc.status, got, tc.flagged)
			}
			v := projectResponse(p)
			if v.Flagged != tc.flagged {
				t.Fatalf("view.Flagged = %v, want %v", v.Flagged, tc.flagged)
			}
		})
	}
}

// Exactly-at-threshold is not quiet: 14 days since the last touch still
// counts as heard-from (the flag means "more than 14 days of silence").
func TestProjectQuietFlaggedThresholdBoundary(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	exactly := now.Add(-quietAfter)
	p := store.ProjectWithLatest{Project: store.Project{Status: "active"}, LatestUpdateAt: &exactly}
	if projectQuietFlagged(p, now) {
		t.Fatal("exactly 14 days old must not be flagged")
	}
	justOver := exactly.Add(-time.Second)
	p.LatestUpdateAt = &justOver
	if !projectQuietFlagged(p, now) {
		t.Fatal("14 days + 1s old must be flagged")
	}
}
