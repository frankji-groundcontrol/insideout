package api

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// heldConn is one Server's guard state: the pinned, pool-less connection that
// holds the lock, and the retry loop that is still trying for it. A struct
// (not a bare *pgx.Conn) so a zero value in the map means "started, holding
// nothing" (storeless tests, a lock that is still taken).
type heldConn struct {
	mu     sync.Mutex
	conn   *pgx.Conn          // set once the lock is won
	cancel context.CancelFunc // ends the retry loop; nil when none runs
	done   chan struct{}      // closed when the retry loop has returned
}

// singletonLockKey names the advisory lock that marks the one api server
// allowed to serve a database. Session-scoped, held on a hijacked connection
// for the process lifetime: while one process holds it, any other process's
// try-lock fails and that process says so. Separate from insideout:migrate
// (store/migrate_role.go), which only serializes migration runs. Addresses
// review-3 SEC-03/REL-05.
const singletonLockKey = "insideout:singleton"

// singletonAcquireTimeout bounds one attempt: the wait for a pool connection
// and the lock query. The first attempt runs from Handler() construction —
// startup, with a fresh pool — so hitting this means something is already
// wrong; the guard stays advisory and gives up rather than block boot.
const singletonAcquireTimeout = 5 * time.Second

// A rolling deploy starts the new process beside the old one, which keeps the
// lock until its 30s shutdown window ends, so finding the lock held is first a
// pause and only later a verdict (rv4-REL-04): the guard retries every
// singletonRetryEvery and reports a second replica only after
// singletonRetryWindow. Two minutes cover the old process's shutdown plus a
// slow health check; a genuine second replica is reported after the same two
// minutes. Vars only so tests can shorten them.
var (
	singletonRetryEvery  = time.Second
	singletonRetryWindow = 2 * time.Minute
)

// singletonGuards marks Servers whose guard already started and pins each
// held connection. Server cannot grow fields without touching server.go, and
// tests build many Servers per process, so the once-per-Server flag lives in
// this process-wide map.
var singletonGuards sync.Map // map[*Server]*heldConn

// startSingletonGuard makes the single-replica contract loud instead of
// silent: the in-memory auth throttle, resend cooldown, turn lock and SSE
// fan-out are only correct with exactly one server process, so at startup
// (withAuthThrottle is constructed once, when main builds the handler) the
// server states that contract once and checks for a second process by trying
// the advisory lock. Finding it held starts a background retry: it takes the
// lock over when the holder exits (a rolling deploy), and logs the ERROR only
// if the holder is still there when the window ends. Advisory only: it never
// blocks startup or serving. The holder keeps the lock until its process exits
// (Postgres ends the session then); tests release it with stopSingletonGuard
// so later tests start from a clean lock.
func (s *Server) startSingletonGuard() {
	if s.log == nil {
		return
	}
	g := &heldConn{}
	if _, dup := singletonGuards.LoadOrStore(s, g); dup {
		return
	}
	s.log.Info("single-replica contract: auth throttle, email resend cooldown, conversation turn lock and SSE fan-out are per-process — run exactly one api server replica (docs/usage/deployment.md)")
	if s.store == nil {
		return // handler tests without a database; nothing to check against
	}
	ok, err := s.trySingletonLock(context.Background(), g)
	if err != nil {
		s.log.Warn("singleton guard: cannot check for a second api server process", "error", err)
		return
	}
	if ok {
		return
	}
	every, window := singletonRetryEvery, singletonRetryWindow // read here, on the caller's goroutine
	s.log.Warn("singleton guard: another api server process holds the singleton lock — normal for a moment while a rolling deploy replaces it; retrying, and reporting a second replica only if it is still there when the window ends (docs/usage/deployment.md)", "retry_every", every, "window", window)
	parent := s.streamsCtx // shutdown ends the retry; a Server built as a literal has none
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, window)
	g.mu.Lock()
	g.cancel, g.done = cancel, make(chan struct{})
	g.mu.Unlock()
	go s.retrySingletonLock(ctx, cancel, g, every, window)
}

// trySingletonLock makes one attempt at the lock. Hijack the connection out of
// the pool: a merely acquired one would make the pool's Close wait for it
// forever, hanging shutdown. Hijacked, it sits outside the pool budget, so the
// pool still serves its full size for requests and the held lock adds one
// connection per replica. Won: the connection is kept in g. Refused: it is
// closed, so a waiting process holds no connection between attempts.
func (s *Server) trySingletonLock(ctx context.Context, g *heldConn) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, singletonAcquireTimeout)
	defer cancel()
	acquired, err := s.store.Pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	held := acquired.Hijack()
	var ok bool
	if err := held.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtext($1))`, singletonLockKey).Scan(&ok); err != nil {
		held.Close(context.Background())
		return false, err
	}
	if !ok {
		held.Close(context.Background())
		return false, nil
	}
	g.mu.Lock()
	g.conn = held
	g.mu.Unlock()
	return true, nil
}

// retrySingletonLock polls for the lock until it wins it, the window ends or
// the guard is stopped. An error (a closed pool at shutdown, a database that
// went away) ends the loop quietly: the guard is advisory.
func (s *Server) retrySingletonLock(ctx context.Context, cancel context.CancelFunc, g *heldConn, every, window time.Duration) {
	defer close(g.done)
	defer cancel()
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				s.log.Error("singleton guard: another api server process already holds the singleton lock — a second replica silently loosens every per-process limit (auth throttle, resend cooldown, turn lock, SSE fan-out); keep exactly one replica (docs/usage/deployment.md)", "after", window)
			}
			return
		case <-tick.C:
			ok, err := s.trySingletonLock(ctx, g)
			if ok {
				s.log.Info("singleton guard: took over the singleton lock after the previous api server process exited")
				return
			}
			if err != nil && ctx.Err() == nil {
				s.log.Debug("singleton guard: retry stopped", "error", err)
				return
			}
		}
	}
}

// stopSingletonGuard ends this Server's retry and closes its held connection,
// releasing the singleton lock. Only tests call it; a running server ends the
// session (and so releases the lock) when the process exits.
func (s *Server) stopSingletonGuard() {
	v, ok := singletonGuards.LoadAndDelete(s)
	if !ok {
		return
	}
	g := v.(*heldConn)
	g.mu.Lock()
	cancel, done := g.cancel, g.done
	g.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		g.conn.Close(context.Background())
		g.conn = nil
	}
}
