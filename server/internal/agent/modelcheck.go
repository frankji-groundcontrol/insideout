package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CheckModel calls GET /v1/models and reports whether a.model is in the
// list — a loud startup signal instead of BUG-009's silent
// model_not_found surfacing only at the first real user request.
// Gateways sometimes omit or misreport this endpoint, so a request
// error is not fatal — the caller logs and continues either way.
func (a *AnthropicStreamer) CheckModel(ctx context.Context) (ok bool, available []string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/v1/models", nil)
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("x-api-key", a.authToken)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("agent: GET /v1/models returned %s", resp.Status)
	}

	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, nil, err
	}
	for _, m := range body.Data {
		available = append(available, m.ID)
		if m.ID == a.model {
			ok = true
		}
	}
	return ok, available, nil
}
