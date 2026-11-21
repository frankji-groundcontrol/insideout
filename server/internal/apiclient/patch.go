package apiclient

// ClearWord is the value that clears a field on every patch verb of the CLI
// and the MCP (docs/usage/cli-mcp-and-guidance.md, "Absent, empty and clear").
const ClearWord = "clear"

// PatchText is the one rule the CLI flags and the MCP arguments follow for a
// field that can be cleared: an absent or empty value leaves it unchanged
// (nil), and the word clear clears it (a pointer to "").
func PatchText(v string) *string {
	switch v {
	case "":
		return nil
	case ClearWord:
		v = ""
	}
	return &v
}

// NonEmpty is a field that cannot be cleared (a title, a status): nil when v is
// empty, else v.
func NonEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
