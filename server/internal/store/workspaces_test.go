package store

import (
	"regexp"
	"testing"
)

// R2: the invite code is the SOLE credential for joining a workspace, and the
// join endpoint is authenticated but not rate-limited — so it must be
// unguessable. The old `%06d` produced a 10^6 keyspace that was
// brute-forceable. The code is now 128 bits from crypto/rand, hex-encoded
// (32 lowercase hex chars). This test pins that format AND proves the keyspace
// is large enough that even 2000 draws never collide (a 10^6 space collides
// almost immediately — the birthday bound is ~1.2k).
var inviteCodeRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestGenerateInviteCode_KeySpace(t *testing.T) {
	const draws = 2000
	seen := make(map[string]struct{}, draws)
	for i := 0; i < draws; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("generateInviteCode: %v", err)
		}
		if !inviteCodeRe.MatchString(code) {
			t.Fatalf("generateInviteCode() = %q, want 32 lowercase hex chars (128 bits)", code)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("generateInviteCode() collided on %q after %d draws — keyspace too small", code, i)
		}
		seen[code] = struct{}{}
	}
}
