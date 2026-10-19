package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/jackc/pgx/v5"
)

// rv4-COMP-05: two links of one repo that run at the same moment each look for
// a holder before either has committed, and each finds none. SetProjectRepo
// takes a transaction-scoped advisory lock on the repo (lower-cased, so any
// spelling) to make the second look wait for the first commit.
//
// The in-process race test in the api package cannot be trusted to notice a
// missing lock: with the lock removed it still passed 60 runs out of 60, the
// window being too narrow. This test holds the first link open instead. It has
// taken the lock and written its row but not committed, which is exactly the
// moment the race needs, and the second link must wait for it and then find the
// repo taken.
func TestSetProjectRepoWaitsForALinkOfTheSameRepoThatIsInFlight(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	first, second := mkTenant(t, st), mkTenant(t, st)
	owner, repo := uniqueSlug()
	url := github.CanonicalRepoURL(owner, repo)

	inFlight, release, held := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		held <- st.withUserContext(ctx, first.user.ID, func(tx pgx.Tx) error {
			// The key is claimRepo's contract; upper case proves it ignores case.
			if _, err := tx.Exec(ctx,
				`SELECT pg_advisory_xact_lock(hashtext('insideout.repo_binding'), hashtext(lower($1)))`, strings.ToUpper(url),
			); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE insideout.projects SET repo_url = $2 WHERE id = $1`, first.project.ID, url); err != nil {
				return err
			}
			close(inFlight)
			<-release
			return nil
		})
	}()
	select {
	case <-inFlight:
	case err := <-held:
		t.Fatalf("in-flight link did not start: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := st.SetProjectRepo(ctx, second.user.ID, second.project.ID, url)
		result <- err
	}()

	var got error
	returned := false
	select {
	case got = <-result:
		returned = true
		t.Errorf("SetProjectRepo returned (%v) while another link of the same repo was uncommitted, want it to wait for that link", got)
	case <-time.After(400 * time.Millisecond):
	}

	close(release)
	if err := <-held; err != nil {
		t.Fatalf("in-flight link: %v", err)
	}
	if !returned {
		got = <-result
	}
	if !errors.Is(got, ErrRepoTaken) {
		t.Errorf("link that waited: err = %v, want ErrRepoTaken once the first link committed", got)
	}
}
