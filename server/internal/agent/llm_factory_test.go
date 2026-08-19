package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLLMStreamerMessagesVsResponsesPaths(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n")
	}))
	defer srv.Close()

	resp, err := NewLLMStreamer(srv.URL+"/v1", "k", "m", "responses")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = resp.StreamChat(t.Context(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if path != "/v1/responses" {
		t.Fatalf("responses path = %q, want /v1/responses", path)
	}

	msg, err := NewLLMStreamer(srv.URL+"/v1", "k", "m", "messages")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = msg.StreamChat(t.Context(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil)
	if path != "/v1/messages" {
		t.Fatalf("messages path = %q, want /v1/messages", path)
	}
}
