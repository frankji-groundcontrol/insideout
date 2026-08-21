package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"zen":"Design for failure."}`)
	if !VerifyWebhookSignature("s3cret", body, sign("s3cret", body)) {
		t.Error("valid signature rejected")
	}
	if VerifyWebhookSignature("s3cret", body, sign("other", body)) {
		t.Error("wrong secret accepted")
	}
	if VerifyWebhookSignature("s3cret", append(body, 'x'), sign("s3cret", body)) {
		t.Error("tampered body accepted")
	}
	if VerifyWebhookSignature("s3cret", body, "md5=abc") {
		t.Error("wrong scheme accepted")
	}
	if VerifyWebhookSignature("s3cret", body, "sha256=not-hex!") {
		t.Error("malformed hex accepted")
	}
}

func TestWebhookRepository(t *testing.T) {
	name, err := WebhookRepository([]byte(`{"repository":{"full_name":"frankji-groundcontrol/insideout"}}`))
	if err != nil || name != "frankji-groundcontrol/insideout" {
		t.Fatalf("got %q, err %v", name, err)
	}
	if _, err := WebhookRepository([]byte(`{}`)); err == nil {
		t.Error("missing full_name accepted")
	}
	if _, err := WebhookRepository([]byte(`not json`)); err == nil {
		t.Error("invalid JSON accepted")
	}
}

func TestRepoURLFor(t *testing.T) {
	if got := RepoURLFor("a/b"); got != "https://github.com/a/b" {
		t.Fatalf("got %q", got)
	}
}
