package agent

import "fmt"

// NewLLMStreamer picks the wire format from schema. chat URL is
// {base}/{schema}; base already includes any /v1.
func NewLLMStreamer(baseURL, apiKey, model, schema string) (ChatStreamer, error) {
	switch schema {
	case "responses":
		return NewResponsesStreamer(baseURL, apiKey, model)
	case "messages", "":
		return NewAnthropicStreamer(baseURL, apiKey, model)
	default:
		return nil, fmt.Errorf("agent: unknown llm schema %q", schema)
	}
}
