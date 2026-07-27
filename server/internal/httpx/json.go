// Package httpx holds small JSON request/response helpers shared by every
// handler, keeping the error contract in one place.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// WriteJSON writes v as a JSON response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ErrorBody is the `{ "error": string, "code"?: string, ... }` contract
// documented in docs/plans/2026-07-20-go-rewrite/02-backend-go.md §3.
// Extra holds additional fields merged into the top-level JSON object
// (e.g. retry_after_seconds) — kept snake_case to match the preserved AI
// throttling contract verbatim.
type ErrorBody struct {
	Error string
	Code  string
	Extra map[string]interface{}
}

func WriteError(w http.ResponseWriter, status int, message string, code string, extra map[string]interface{}) {
	body := map[string]interface{}{"error": message}
	if code != "" {
		body["code"] = code
	}
	for k, v := range extra {
		body[k] = v
	}
	WriteJSON(w, status, body)
}

// DecodeJSON decodes the request body into v, rejecting unknown fields so
// client typos surface as 400s instead of silently-ignored data. It also
// rejects trailing bytes after the first JSON value (`{}{}`, `{} garbage`):
// Decoder.Decode stops at the first value and would otherwise silently drop
// whatever follows, defeating the strictness DisallowUnknownFields asks for.
// A clean body has exactly one value, so the next Decode must hit io.EOF
// (R3). An empty body still returns io.EOF from the FIRST Decode — callers
// that allow a body-less POST (e.g. create-revision) check for that.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("httpx: trailing data after JSON body")
	}
	return nil
}
