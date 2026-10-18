package api

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
	atThreshold := now.Add(-quietAfter)
	overThreshold := atThreshold.Add(-time.Second)
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
		{"update exactly at the threshold is not flagged", "active", 30 * 24 * time.Hour, &atThreshold, &k, &c, false},
		{"update a second over the threshold is flagged", "active", 30 * 24 * time.Hour, &overThreshold, &k, &c, true},
		{"creation exactly at the threshold is not flagged", "planning", quietAfter, nil, nil, nil, false},
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

// FS-09: the board list sent every project's whole latest note (bounded only
// by maxUpdateContentRunes, 5000 runes) while a card shows one ellipsized
// line. The board response now carries a short preview; the full text stays
// in the updates feed.
func TestProjectResponseBoardNotePreview(t *testing.T) {
	preview := func(content string) string {
		got := projectResponse(store.ProjectWithLatest{LatestUpdateContent: &content}).LatestUpdateContent
		if got == nil {
			t.Fatal("a present note must keep a preview")
		}
		return *got
	}

	t.Run("a long multibyte note is cut at a rune boundary", func(t *testing.T) {
		got := preview(strings.Repeat("世", maxUpdateContentRunes))
		if !utf8.ValidString(got) {
			t.Fatalf("preview is not valid UTF-8: %q", got)
		}
		if n := utf8.RuneCountInString(got); n > 201 {
			t.Fatalf("preview has %d runes, want at most 201 (200 + ellipsis)", n)
		}
		if want := strings.Repeat("世", 200) + "…"; got != want {
			t.Fatalf("preview = %q, want the first 200 runes plus an ellipsis", got)
		}
	})

	t.Run("one rune over the preview is cut, exactly at it is not", func(t *testing.T) {
		exact := strings.Repeat("a", 200)
		if got := preview(exact); got != exact {
			t.Fatalf("a 200-rune note must pass unchanged, got %q", got)
		}
		if got := preview(exact + "a"); got != exact+"…" {
			t.Fatalf("a 201-rune note must be cut to 200 + ellipsis, got %q", got)
		}
	})

	t.Run("a short note passes through and nil stays nil", func(t *testing.T) {
		short := "shipped the invitation flow"
		if got := preview(short); got != short {
			t.Fatalf("a short note must pass unchanged, got %q", got)
		}
		if got := projectResponse(store.ProjectWithLatest{}).LatestUpdateContent; got != nil {
			t.Fatalf("a missing note must stay nil, got %q", *got)
		}
	})
}
