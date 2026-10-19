package apiclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// IdeaCreate captures a new idea (POST /workspaces/{id}/ideas).
func (c *Client) IdeaCreate(workspaceID, title, content string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("workspaces", workspaceID, "ideas"),
		map[string]string{"title": title, "content": content}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IdeaConvert turns an idea into a PRD + coach conversation
// (POST /ideas/{id}/convert). The optional locale is the language the coach
// speaks ("en" or "zh", see ConvertLocale); without one the body names none
// and the server keeps its default (zh). It is a trailing variadic rather than
// a second method so the escape sweep keeps one row per URL shape.
func (c *Client) IdeaConvert(ideaID string, locale ...string) (json.RawMessage, error) {
	body := map[string]any{}
	if len(locale) > 0 && locale[0] != "" {
		body["locale"] = locale[0]
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("ideas", ideaID, "convert"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ConvertLocale picks the language a convert request asks the coach to speak.
// explicit (the --locale flag or the tool's locale argument, "" when absent)
// must be en or zh, written plainly or as a locale tag such as en-US. With
// none, the first of LC_ALL, LC_MESSAGES and LANG that is set decides (zh is
// zh) and everything else, or no variable at all, is en. The narrowing
// happens here because the server turns every tag that is not English into
// zh: a typo or an unsupported language would otherwise open a Chinese
// conversation.
func ConvertLocale(explicit string) (string, error) {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		if l := languageOf(explicit); l == "en" || l == "zh" {
			return l, nil
		}
		return "", fmt.Errorf("locale %q is not supported: use en or zh", explicit)
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			if languageOf(v) == "zh" {
				return "zh", nil
			}
			break // the first variable that is set decides
		}
	}
	return "en", nil
}

// languageOf is the primary language of a locale tag, lower-cased:
// "zh_CN.UTF-8" and "zh-Hans" are both "zh".
func languageOf(tag string) string {
	if i := strings.IndexAny(tag, "_-.@"); i >= 0 {
		tag = tag[:i]
	}
	return strings.ToLower(tag)
}

// Prd returns GET /prds/{id}.
func (c *Client) Prd(id string) (json.RawMessage, error) { return c.get(apiPath("prds", id)) }

// PrdRevisions lists a PRD's revision snapshots (GET /prds/{id}/revisions).
func (c *Client) PrdRevisions(prdID string) (json.RawMessage, error) {
	return c.get(apiPath("prds", prdID, "revisions"))
}

// SnapshotPrd records a revision snapshot with an optional note
// (POST /prds/{id}/revisions).
func (c *Client) SnapshotPrd(prdID, note string) (json.RawMessage, error) {
	body := map[string]any{}
	if note != "" {
		body["note"] = note
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("prds", prdID, "revisions"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CommitPrd performs the human Commit: freeze the working PRD as an
// immutable named version (POST /prds/{id}/commit).
func (c *Client) CommitPrd(prdID, name, audience, summary string, unresolved []string, decisionNote string) (json.RawMessage, error) {
	if unresolved == nil {
		unresolved = []string{}
	}
	body := map[string]any{
		"name": name, "primaryAudience": audience, "changeSummary": summary,
		"unresolved": unresolved, "decisionNote": decisionNote,
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("prds", prdID, "commit"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PrdVersions lists a PRD's commits, newest first
// (GET /prds/{id}/commits).
func (c *Client) PrdVersions(prdID string) (json.RawMessage, error) {
	return c.get(apiPath("prds", prdID, "commits"))
}

// PrdReadiness returns the per-audience gap disclosure
// (GET /prds/{id}/readiness) for "form a version now".
func (c *Client) PrdReadiness(prdID string) (json.RawMessage, error) {
	return c.get(apiPath("prds", prdID, "readiness"))
}

// PrdView returns an audience's projection of the PRD core
// (GET /prds/{id}/view?audience=…).
func (c *Client) PrdView(prdID, audience string) (json.RawMessage, error) {
	return c.get(withQuery(apiPath("prds", prdID, "view"), "audience", audience))
}

// PrdExport downloads a PRD export (GET /prds/{id}/export) as raw bytes.
// format is "markdown" (default) or "print" (HTML); audience is optional
// and only valid with markdown.
func (c *Client) PrdExport(prdID, format, audience string) ([]byte, error) {
	if format == "" {
		format = "markdown"
	}
	q := url.Values{"format": {format}}
	if audience != "" {
		q.Set("audience", audience)
	}
	return c.getBytes(apiPath("prds", prdID, "export") + "?" + q.Encode())
}

// PrdExportAudience returns an audience-projected markdown export
// (GET /prds/{id}/export?format=markdown&audience=…) as raw bytes.
func (c *Client) PrdExportAudience(prdID, audience string) ([]byte, error) {
	return c.getBytes(withQuery(apiPath("prds", prdID, "export"), "format", "markdown", "audience", audience))
}

// AgentContext returns the compact, focus-scoped agent context
// (GET /agent/context).
func (c *Client) AgentContext(projectID, mode, focus string) (json.RawMessage, error) {
	kv := []string{"project_id", projectID}
	if mode != "" {
		kv = append(kv, "mode", mode)
	}
	if focus != "" {
		kv = append(kv, "focus", focus)
	}
	return c.get(withQuery(apiPath("agent", "context"), kv...))
}

// AgentCheckpoint records agent work (POST /agent/checkpoint).
func (c *Client) AgentCheckpoint(projectID, nodeID, summary string) (json.RawMessage, error) {
	body := map[string]any{"projectId": projectID, "summary": summary}
	if nodeID != "" {
		body["nodeId"] = nodeID
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/agent/checkpoint", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ProposeItem is one structured add_node entry of an agent proposal; ParentHint
// names the parent node by title. Kept as two fields, not a "Title@Parent"
// string, so a title may contain "@".
type ProposeItem struct{ Title, ParentHint string }

// AgentPropose records an agent proposal (POST /agent/propose).
// items: optional structured entries a human can apply on acceptance.
func (c *Client) AgentPropose(projectID, kind, summary, detail string, items []ProposeItem) (json.RawMessage, error) {
	body := map[string]any{"projectId": projectID, "kind": kind, "summary": summary, "detail": detail}
	if len(items) > 0 {
		arr := make([]map[string]string, 0, len(items))
		for _, it := range items {
			arr = append(arr, map[string]string{"action": "add_node", "title": it.Title, "parentHint": it.ParentHint})
		}
		body["items"] = arr
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/agent/propose", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DecideProposal records the human accept/reject of an agent proposal
// (POST /agent/proposals/{id}/decision); apply executes structured
// items on acceptance.
func (c *Client) DecideProposal(updateID, decision, reason string, apply bool) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("agent", "proposals", updateID, "decision"),
		map[string]any{"decision": decision, "reason": reason, "apply": apply}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
