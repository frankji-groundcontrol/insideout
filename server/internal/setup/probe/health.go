package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Check is one line of a service health check.
type Check struct {
	Name   string
	Status string // ok or fail
	Detail string
}

// ServiceHealth checks that baseURL is an InsideOut service that is fit to
// use. Reaching the host is not enough: /healthz must report ok (it pings
// the database), and the API must answer /me.
func ServiceHealth(ctx context.Context, client *http.Client, baseURL string) ([]Check, error) {
	if client == nil {
		client = defaultClient()
	}
	origin, err := serviceOrigin(baseURL)
	if err != nil {
		return []Check{{Name: "address", Status: "fail", Detail: err.Error()}}, err
	}
	checks := []Check{{Name: "address", Status: "ok", Detail: origin}}

	health, herr := checkHealthz(ctx, client, origin+"/healthz")
	checks = append(checks, health)
	api, aerr := checkAPIDoor(ctx, client, baseURL)
	checks = append(checks, api)

	switch {
	case herr != nil:
		return checks, herr
	case aerr != nil:
		return checks, aerr
	default:
		return checks, nil
	}
}

func serviceOrigin(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("the address must be an http(s) URL with a host")
	}
	return u.Scheme + "://" + u.Host, nil
}

func checkHealthz(ctx context.Context, client *http.Client, healthURL string) (Check, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return Check{Name: "health", Status: "fail", Detail: err.Error()}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Check{Name: "health", Status: "fail", Detail: "could not reach /healthz"}, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode == http.StatusOK && healthBodyOK(body) {
		return Check{Name: "health", Status: "ok", Detail: "/healthz ok"}, nil
	}
	detail := healthDetail(resp.StatusCode, body)
	return Check{Name: "health", Status: "fail", Detail: detail}, fmt.Errorf("%s", detail)
}

// healthBodyOK accepts the current JSON report and the older plain "ok".
func healthBodyOK(body []byte) bool {
	text := strings.TrimSpace(string(body))
	if strings.EqualFold(text, "ok") {
		return true
	}
	var payload struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(body, &payload) == nil && strings.EqualFold(payload.Status, "ok")
}

func healthDetail(code int, body []byte) string {
	snippet := strings.Join(strings.Fields(string(body)), " ")
	if len(snippet) > 80 {
		snippet = snippet[:80] + "…"
	}
	if snippet == "" {
		return fmt.Sprintf("/healthz answered %d with an empty body", code)
	}
	return fmt.Sprintf("/healthz answered %d: %s", code, snippet)
}

func checkAPIDoor(ctx context.Context, client *http.Client, baseURL string) (Check, error) {
	res, err := APIProbe(ctx, client, baseURL)
	switch {
	case err != nil && !res.Reachable:
		return Check{Name: "api", Status: "fail", Detail: "could not reach the API"}, err
	case err != nil:
		return Check{Name: "api", Status: "fail", Detail: err.Error()}, err
	case res.AuthRequired:
		return Check{Name: "api", Status: "ok", Detail: "authentication required"}, nil
	default:
		return Check{Name: "api", Status: "ok", Detail: "answered"}, nil
	}
}
