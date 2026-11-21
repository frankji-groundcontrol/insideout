package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Rate limiter + circuit breaker: thin wrappers over the SQL functions in
// db/migrations/..._ai_ops.sql.

type RateLimitResult struct {
	Allowed           bool
	CurrentCount      int
	MaxRequests       int
	RetryAfterSeconds int
}

func (s *Store) CheckRateLimit(ctx context.Context, userID uuid.UUID) (*RateLimitResult, error) {
	var res RateLimitResult
	err := s.withUserContext(ctx, userID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT allowed, current_count, max_requests, retry_after_seconds FROM insideout.ai_check_rate_limit($1)`, userID).
			Scan(&res.Allowed, &res.CurrentCount, &res.MaxRequests, &res.RetryAfterSeconds)
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

type CircuitResult struct {
	Allowed           bool
	State             string
	RetryAfterSeconds int
}

func (s *Store) CheckCircuit(ctx context.Context) (*CircuitResult, error) {
	var res CircuitResult
	err := s.Pool.QueryRow(ctx, `SELECT allowed, state, retry_after_seconds FROM insideout.ai_check_circuit()`).
		Scan(&res.Allowed, &res.State, &res.RetryAfterSeconds)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Store) RecordCircuitResult(ctx context.Context, success bool) error {
	_, err := s.Pool.Exec(ctx, `SELECT insideout.ai_record_circuit_result($1)`, success)
	return err
}
