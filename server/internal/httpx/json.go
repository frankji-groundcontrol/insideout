// Package httpx holds small JSON request/response helpers shared by every
// handler, keeping the error contract in one place.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// WriteJSON writes v as a JSON response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ErrorBody is the `{ "error": string, "code": string, ... }` contract
// (docs/architecture/backend.md). Code is lowercase snake_case; Extra holds
// additional fields merged into the top-level JSON object, camelCase like
// every other body (e.g. retryAfterSeconds, rv5-PARC-08).
type ErrorBody struct {
	Error string
	Code  string
	Extra map[string]interface{}
}

func WriteError(w http.ResponseWriter, status int, message string, code string, extra map[string]interface{}) {
	if code == "" {
		// Every error carries a code: without a specific one, the status's
		// own name ("not_found", "internal_server_error").
		code = strings.ToLower(strings.ReplaceAll(http.StatusText(status), " ", "_"))
	}
	body := map[string]interface{}{"error": message, "code": code}
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
