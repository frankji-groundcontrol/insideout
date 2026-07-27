package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// TestProjectUpdates_Pagination verifies the F9 fix: ListProjectUpdates
// returns one bounded page at a time via a (created_at, id) keyset cursor,
// instead of the whole unbounded history that used to ride inside every
// GetProject response. The load-bearing properties are the cursor's: walking
// page by page must visit every row exactly once (no duplicates, no skipped
// rows) in descending time order, and must terminate.
func TestProjectUpdates_Pagination(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	owner := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, owner.ID, "Paginate WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, owner.ID, ws.ID, "Paginate Proj", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	const total = 5
	ids := make(map[string]bool)
	for i := 0; i < total; i++ {
		u, err := st.AddProjectUpdate(ctx, owner.ID, proj.ID, "note", fmt.Sprintf("update-%d", i))
		if err != nil {
			t.Fatalf("add update %d: %v", i, err)
		}
		ids[u.ID.String()] = true
	}

	// limit<=0 falls back to the default page, which must not truncate a
	// small history — GetProject still embeds the newest page whole.
	all, err := st.ListProjectUpdates(ctx, owner.ID, proj.ID, 0, nil)
	if err != nil {
		t.Fatalf("list default page: %v", err)
	}
	if len(all) != total {
		t.Fatalf("default page = %d rows, want %d (small history must not be truncated)", len(all), total)
	}

	// Walk the cursor two rows at a time and reassemble the full history.
	const pageSize = 2
	var walked []ProjectUpdate
	var before *uuid.UUID
	for pages := 0; ; pages++ {
		if pages > total { // backstop: a broken cursor must not loop forever
			t.Fatalf("pagination did not terminate after %d pages", pages)
		}
		page, err := st.ListProjectUpdates(ctx, owner.ID, proj.ID, pageSize, before)
		if err != nil {
			t.Fatalf("list page %d: %v", pages, err)
		}
		if len(page) == 0 {
			break
		}
		walked = append(walked, page...)
		if len(page) < pageSize {
			break // short page == last page
		}
		last := page[len(page)-1].ID
		before = &last
	}

	// Every row exactly once — the thing an off-by-one cursor gets wrong.
	if len(walked) != total {
		t.Fatalf("walked %d rows across pages, want %d (cursor duplicated or skipped rows)", len(walked), total)
	}
	seen := make(map[uuid.UUID]bool, total)
	for _, u := range walked {
		if seen[u.ID] {
			t.Fatalf("row %s appeared on more than one page", u.ID)
		}
		seen[u.ID] = true
		if !ids[u.ID.String()] {
			t.Fatalf("walked an unexpected row %s", u.ID)
		}
	}

	// Descending time order holds across page boundaries, not just within a page.
	for i := 1; i < len(walked); i++ {
		if walked[i-1].CreatedAt.Before(walked[i].CreatedAt) {
			t.Fatalf("rows out of DESC order across pages: %s (%s) before %s (%s)",
				walked[i-1].ID, walked[i-1].CreatedAt, walked[i].ID, walked[i].CreatedAt)
		}
	}
}
