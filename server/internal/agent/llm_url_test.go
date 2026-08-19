package agent

import "testing"

func TestLLMChatURLDoesNotInsertV1(t *testing.T) {
	got := LLMChatURL("https://gateway.example/v1", "messages")
	if got != "https://gateway.example/v1/messages" {
		t.Fatalf("messages URL = %q", got)
	}
	got = LLMChatURL("https://gateway.example/v1/", "responses")
	if got != "https://gateway.example/v1/responses" {
		t.Fatalf("responses URL = %q", got)
	}
}

func TestLLMModelsURLIsSiblingOfSchema(t *testing.T) {
	got := LLMModelsURL("https://gateway.example/v1")
	if got != "https://gateway.example/v1/models" {
		t.Fatalf("models URL = %q", got)
	}
}
