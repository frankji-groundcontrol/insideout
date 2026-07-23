package auth

import "testing"

func TestGenerateRefreshToken_HashConsistencyAndUniqueness(t *testing.T) {
	token1, hash1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if HashRefreshToken(token1) != hash1 {
		t.Fatal("HashRefreshToken(token) must match the hash returned alongside it")
	}

	token2, hash2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if token1 == token2 || hash1 == hash2 {
		t.Fatal("two generated refresh tokens must be distinct")
	}
}
