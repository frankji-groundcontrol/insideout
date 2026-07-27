package httpx

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

type sample struct {
	Name string `json:"name"`
}

func decodeBody(t *testing.T, body string) error {
	t.Helper()
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	var v sample
	return DecodeJSON(r, &v)
}

// R3: DecodeJSON must accept exactly one JSON value and nothing else. A clean
// body decodes; an empty body returns io.EOF (callers that allow a body-less
// POST check for it); unknown fields and any trailing bytes after the first
// value are rejected instead of silently ignored.
func TestDecodeJSON(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr error // nil = expect success; io.EOF / non-nil sentinel otherwise
	}{
		{"clean object", `{"name":"x"}`, nil},
		{"trailing whitespace only", `{"name":"x"}  ` + "\n", nil},
		{"empty body is io.EOF", ``, io.EOF},
		{"unknown field rejected", `{"name":"x","bogus":1}`, nil}, // non-nil, not io.EOF
		{"second value rejected", `{"name":"x"}{"name":"y"}`, nil},
		{"trailing garbage rejected", `{"name":"x"} garbage`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := decodeBody(t, tc.body)
			switch tc.name {
			case "clean object", "trailing whitespace only":
				if err != nil {
					t.Fatalf("DecodeJSON(%q) = %v, want nil", tc.body, err)
				}
			case "empty body is io.EOF":
				if !errors.Is(err, io.EOF) {
					t.Fatalf("DecodeJSON(%q) = %v, want io.EOF", tc.body, err)
				}
			default: // all the rejection cases
				if err == nil {
					t.Fatalf("DecodeJSON(%q) = nil, want an error", tc.body)
				}
			}
		})
	}
}
