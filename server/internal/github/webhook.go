package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// VerifyWebhookSignature checks GitHub's X-Hub-Signature-256 header
// ("sha256=<hex hmac of the raw body>") against the shared secret, in
// constant time.
func VerifyWebhookSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

// WebhookRepository extracts "owner/name" from a delivery payload's
// repository.full_name (push, pull_request, and most other events share
// the shape).
func WebhookRepository(payload []byte) (string, error) {
	var p struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return "", err
	}
	name := strings.TrimSpace(p.Repository.FullName)
	if name == "" || strings.Contains(name, "/") == false {
		return "", ErrRepoNotFound
	}
	return name, nil
}

// RepoURLFor turns "owner/name" into the canonical repo URL form the
// projects table stores (what PUT /projects/{id}/repo accepted).
func RepoURLFor(ownerSlashName string) string {
	return "https://github.com/" + ownerSlashName
}
