package agent

import "strings"

// LLMChatURL joins the operator-supplied base (which already includes any
// /v1) with the schema path. The server never inserts "/v1".
func LLMChatURL(base, schema string) string {
	return strings.TrimSuffix(base, "/") + "/" + schema
}

// LLMModelsURL is GET {base}/models — sibling of /messages or /responses.
func LLMModelsURL(base string) string {
	return strings.TrimSuffix(base, "/") + "/models"
}
