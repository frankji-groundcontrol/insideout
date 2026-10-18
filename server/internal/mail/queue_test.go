package mail

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// rv4-REL-07 / rv4-SEC-03: Enqueue is what lets an emailed auth request answer
// before the provider does. These tests pin its contract: it returns at once,
// every accepted job runs, concurrency and backlog are bounded, and a bad job
// costs a log line, not the process.

func within(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !cond(); {
		if time.Now().After(deadline) {
			t.Fatalf("gave up waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestEnqueueReturnsBeforeTheJobFinishes(t *testing.T) {
	s := NewSender("", "noreply@example.com", testLogger(), false)
	release, ran := make(chan struct{}), make(chan struct{})
	if !s.Enqueue("held", func(ctx context.Context) error {
		defer close(ran)
		<-release
		return nil
	}) {
		t.Fatal("an empty queue refused a job")
	}
	select {
	case <-ran:
		t.Fatal("the job finished before it was released: Enqueue ran it inline")
	default:
	}
	close(release)
	s.Wait()
	select {
	case <-ran:
	default:
		t.Fatal("Wait returned before the accepted job finished")
	}
}

func TestEnqueueRunsEveryAcceptedJobWithItsOwnDeadline(t *testing.T) {
	s := NewSender("", "noreply@example.com", testLogger(), false)
	var done, bounded atomic.Int64
	const n = queueDepth / 2
	for i := 0; i < n; i++ {
		s.Enqueue("count", func(ctx context.Context) error {
			if dl, ok := ctx.Deadline(); ok && time.Until(dl) <= jobTimeout && ctx.Err() == nil {
				bounded.Add(1)
			}
			done.Add(1)
			return nil
		})
	}
	s.Wait()
	if done.Load() != n || bounded.Load() != n {
		t.Fatalf("ran %d of %d jobs, %d with a live context bounded by jobTimeout", done.Load(), n, bounded.Load())
	}
}

func TestEnqueueNeverRunsMoreThanQueueWorkersAtOnce(t *testing.T) {
	s := NewSender("", "noreply@example.com", testLogger(), false)
	var running, peak atomic.Int64
	release := make(chan struct{})
	for i := 0; i < 3*queueWorkers; i++ {
		s.Enqueue("held", func(ctx context.Context) error {
			now := running.Add(1)
			for old := peak.Load(); now > old && !peak.CompareAndSwap(old, now); old = peak.Load() {
			}
			<-release
			running.Add(-1)
			return nil
		})
	}
	within(t, "the workers to fill up", func() bool { return running.Load() == queueWorkers })
	time.Sleep(20 * time.Millisecond) // room for a worker beyond the cap to show itself
	close(release)
	s.Wait()
	if peak.Load() != queueWorkers {
		t.Fatalf("peak concurrency = %d, want exactly %d", peak.Load(), queueWorkers)
	}
}

func TestEnqueueDropsAndLogsWhenTheBacklogIsFull(t *testing.T) {
	var logs bytes.Buffer
	s := NewSender("", "noreply@example.com", slog.New(slog.NewTextHandler(&logs, nil)), false)
	var started, ran atomic.Int64
	release := make(chan struct{})
	job := func(ctx context.Context) error {
		started.Add(1)
		<-release
		ran.Add(1)
		return nil
	}
	for i := 0; i < queueWorkers; i++ {
		s.Enqueue("held", job)
	}
	within(t, "every worker to hold a job", func() bool { return started.Load() == queueWorkers })
	for i := 0; i < queueDepth; i++ { // the backlog fills exactly
		if !s.Enqueue("held", job) {
			t.Fatalf("job %d of %d was refused with room left", i+1, queueDepth)
		}
	}
	if s.Enqueue("overflow", func(ctx context.Context) error { t.Error("a dropped job ran"); return nil }) {
		t.Fatal("a full queue accepted one more job")
	}
	close(release)
	s.Wait()
	if want := int64(queueWorkers + queueDepth); ran.Load() != want {
		t.Fatalf("ran %d jobs, want the %d accepted", ran.Load(), want)
	}
	if !strings.Contains(logs.String(), "overflow: mail queue full") {
		t.Fatalf("the drop was not logged:\n%s", logs.String())
	}
}

func TestEnqueueSurvivesFailingAndPanickingJobs(t *testing.T) {
	var logs bytes.Buffer
	s := NewSender("", "noreply@example.com", slog.New(slog.NewTextHandler(&logs, nil)), false)
	var after atomic.Int64
	s.Enqueue("broken", func(ctx context.Context) error { return errors.New("provider said no") })
	s.Enqueue("explosive", func(ctx context.Context) error { panic("boom") })
	s.Wait() // the panic is contained: the test binary is still here
	s.Enqueue("fine", func(ctx context.Context) error { after.Add(1); return nil })
	s.Wait()
	if after.Load() != 1 {
		t.Fatal("the queue stopped serving after a failing and a panicking job")
	}
	for _, want := range []string{"broken: send", "provider said no", "explosive: send panicked", "boom"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, logs.String())
		}
	}
}

// An idle sender holds no goroutines, and Wait on an idle sender does not block.
func TestIdleSenderHoldsNoWorkers(t *testing.T) {
	s := NewSender("", "noreply@example.com", testLogger(), false)
	s.Wait()
	s.Enqueue("once", func(ctx context.Context) error { return nil })
	s.Wait()
	within(t, "the worker to exit", func() bool {
		s.qmu.Lock()
		defer s.qmu.Unlock()
		return s.workers == 0
	})
}
