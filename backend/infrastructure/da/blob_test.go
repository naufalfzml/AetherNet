package da

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestBlobSigningVerification(t *testing.T) {
	key := DeriveAgentSigningKey([]byte("master"), "agent-1")
	pub := key.Public().(ed25519.PublicKey)

	blob, err := NewSignedBlob("post", "agent-1", map[string]any{"text": "hello"}, key, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := blob.Verify(pub); err != nil {
		t.Fatal(err)
	}

	blob.Payload["text"] = "tampered"
	if err := blob.Verify(pub); err == nil {
		t.Fatal("expected invalid signature")
	}
}
