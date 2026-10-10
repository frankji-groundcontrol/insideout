package api

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// heldConn is the guard's pinned, pool-less connection; a struct (not a
// bare *pgx.Conn) so a zero value in the map means "started, holding
// nothing" (storeless tests, a refused lock).
type heldConn struct{ conn *pgx.Conn }

// singletonLockKey names the advisory lock that marks the one api server
// allowed to serve a database. Session-scoped, held on a hijacked connection
// for the process lifetime: while one process holds it, any other process's
// try-lock fails and that process says so. Separate from insideout:migrate
// (store/migrate_role.go), which only serializes migration runs. Addresses
// review-3 SEC-03/REL-05.
const singletonLockKey = "insideout:singleton"

// singletonAcquireTimeout bounds the guard's wait for a pool connection. It
// runs from Handler() construction — startup, with a fresh pool — so hitting
// this means something is already wrong; the guard stays advisory and gives
// up rather than block boot.
const singletonAcquireTimeout = 5 * time.Second

// singletonGuards marks Servers whose guard already started and pins each
// held connection. Server cannot grow fields without touching server.go, and
// tests build many Servers per process, so the once-per-Server flag lives in
// this process-wide map.
var singletonGuards sync.Map // map[*Server]heldConn

// startSingletonGuard makes the single-replica contract loud instead of
// silent: the in-memory auth throttle, resend cooldown, turn lock and SSE
// fan-out are only correct with exactly one server process, so at startup
// (withAuthThrottle is constructed once, when main builds the handler) the
// server states that contract once and checks for a second process by trying
// the advisory lock — finding it held is the loud signal a scale-up used to
// lack. Advisory only: it never blocks startup or serving. The holder keeps
// the lock until its process exits (Postgres ends the session then); tests
// release it with stopSingletonGuard so later tests start from a clean lock.
func (s *Server) startSingletonGuard() {
	if s.log == nil {
		return
	}
	if _, dup := singletonGuards.LoadOrStore(s, heldConn{}); dup {
		return
	}
	s.log.Info("single-replica contract: auth throttle, email resend cooldown, conversation turn lock and SSE fan-out are per-process — run exactly one api server replica (docs/usage/deployment.md)")
	if s.store == nil {
		return // handler tests without a database; nothing to check against
	}
	ctx, cancel := context.WithTimeout(context.Background(), singletonAcquireTimeout)
	defer cancel()
	// Hijack the connection out of the pool: a merely acquired one would
	// make the pool's Close wait for it forever, hanging shutdown. Hijacked,
	// it sits outside the pool budget, so the pool still serves its full
	// size for requests and this adds one connection per replica.
	acquired, err := s.store.Pool.Acquire(ctx)
	if err != nil {
		s.log.Warn("singleton guard: cannot check for a second api server process", "error", err)
		return
	}
	held := acquired.Hijack()
	var ok bool
	if err := held.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtext($1))`, singletonLockKey).Scan(&ok); err != nil {
		held.Close(context.Background())
		s.log.Warn("singleton guard: cannot check for a second api server process", "error", err)
		return
	}
	if !ok {
		held.Close(context.Background())
		s.log.Error("singleton guard: another api server process already holds the singleton lock — a second replica silently loosens every per-process limit (auth throttle, resend cooldown, turn lock, SSE fan-out); keep exactly one replica (docs/usage/deployment.md)")
		return
	}
	singletonGuards.Store(s, heldConn{conn: held})
}

// stopSingletonGuard closes this Server's held connection, releasing the
// singleton lock. Only tests call it; a running server ends the session (and
// so releases the lock) when the process exits.
func (s *Server) stopSingletonGuard() {
	if v, ok := singletonGuards.LoadAndDelete(s); ok {
		if held := v.(heldConn).conn; held != nil {
			held.Close(context.Background())
		}
	}
}
