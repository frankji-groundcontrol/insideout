package agent

import (
	"context"
	"strings"
	"testing"
)

// rv4-AIDEV-06: a Responses provider may stream several function_call items in
// one response. The parser kept one id, one name and one argument buffer, so a
// second call overwrote the first's id and name while both calls' arguments ran
// together into one invalid object. The request now forbids parallel calls, and
// the parser keeps each item's arguments apart in case a provider ignores that.

const rCall2 = `{"type":"response.output_item.added","output_index":1,"item":{"id":"fc_2","type":"function_call","call_id":"call_2","name":"set_section","arguments":""}}`

func TestTrustParseResponses_TwoCallsAreNeverMerged(t *testing.T) {
	end := rEnd("response.completed", "completed")
	cases := map[string]string{
		"one after the other": sseData(rCall, rArgs("fc_1", `{"kind":"problem"}`), rCall2, rArgs("fc_2", `{"text":"x"}`), end),
		"interleaved": sseData(rCall, rCall2, rArgs("fc_1", `{"kind":`), rArgs("fc_2", `{"text":`),
			rArgs("fc_1", `"problem"}`), rArgs("fc_2", `"x"}`), end),
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			turn, err := parseResponsesStream(strings.NewReader(stream), nil, nil)
			if err != nil {
				t.Fatalf("parseResponsesStream: %v", err)
			}
			call := turn.ToolCall
			if call == nil {
				t.Fatal("turn.ToolCall = nil, want the first call")
			}
			got := call.ID + "/" + call.Name + "/" + call.Arguments
			if want := `call_1/record_fact/{"kind":"problem"}`; got != want {
				t.Fatalf("call = %q, want only the first call, with only its own arguments: %q", got, want)
			}
		})
	}
}

// A request that offers tools must say one call at a time; one that offers
// none must not carry the flag at all.
func TestTrustResponsesStreamer_RequestForbidsParallelToolCalls(t *testing.T) {
	tool := Tool{Name: "get_prd", Description: "reads the prd", Parameters: map[string]any{"type": "object"}}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}
	calls := map[string]func(s *ResponsesStreamer) error{
		"StreamChat": func(s *ResponsesStreamer) error {
			_, err := s.StreamChat(context.Background(), "sys", msgs, []Tool{tool}, nil)
			return err
		},
		"StreamChatForcingTool": func(s *ResponsesStreamer) error {
			_, err := s.StreamChatForcingTool(context.Background(), "sys", msgs, tool, nil)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			srv, reqs := responsesStreamServer(t, "completed")
			s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
			if err != nil {
				t.Fatal(err)
			}
			if err := call(s); err != nil {
				t.Fatal(err)
			}
			body := <-reqs
			if v, ok := body["parallel_tool_calls"].(bool); !ok || v {
				t.Fatalf("parallel_tool_calls = %v (a bool: %v), want false", body["parallel_tool_calls"], ok)
			}
		})
	}
}

func TestTrustResponsesStreamer_NoToolsCarriesNoParallelFlag(t *testing.T) {
	srv, reqs := responsesStreamServer(t, "completed")
	s, err := NewResponsesStreamer(srv.URL, "test-token", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StreamChat(context.Background(), "sys", []Message{{Role: RoleUser, Content: "hi"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if v, present := (<-reqs)["parallel_tool_calls"]; present {
		t.Fatalf("parallel_tool_calls = %v on a request with no tools, want the key absent", v)
	}
}
