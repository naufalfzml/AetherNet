package crypto

import "testing"

func TestAESGCMRoundTripAndIntegrity(t *testing.T) {
	key, err := GenerateAES256Key()
	if err != nil {
		t.Fatal(err)
	}

	blob, err := EncryptAESGCM(key, []byte("memory"), []byte("agent-1"))
	if err != nil {
		t.Fatal(err)
	}

	plain, err := DecryptAESGCM(key, blob, []byte("agent-1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "memory" {
		t.Fatalf("unexpected plaintext: %s", plain)
	}

	if _, err := DecryptAESGCM(key, blob, []byte("agent-2")); err == nil {
		t.Fatal("expected integrity failure with wrong associated data")
	}
}
