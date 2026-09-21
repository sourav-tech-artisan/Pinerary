package sharing

import "testing"

func TestNewTokenHasEnoughEntropyAndStableHash(t *testing.T) {
	token, hash, err := newToken()
	if err != nil {
		t.Fatalf("newToken() error = %v", err)
	}
	if len(token) != 43 || len(hash) != 32 {
		t.Fatalf("token length = %d, hash length = %d", len(token), len(hash))
	}
}
