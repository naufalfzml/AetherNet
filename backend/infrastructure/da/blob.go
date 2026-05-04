package da

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type Blob struct {
	Type      string         `json:"type"`
	AgentID   string         `json:"agentId"`
	Payload   map[string]any `json:"payload"`
	Sig       string         `json:"sig"`
	Timestamp time.Time      `json:"timestamp"`
}

func NewSignedBlob(eventType, agentID string, payload map[string]any, key ed25519.PrivateKey, timestamp time.Time) (Blob, error) {
	blob := Blob{Type: eventType, AgentID: agentID, Payload: payload, Timestamp: timestamp.UTC()}
	canonical, err := blob.canonicalPayload()
	if err != nil {
		return Blob{}, err
	}
	blob.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(key, canonical))
	return blob, nil
}

func (b Blob) Verify(pub ed25519.PublicKey) error {
	sig, err := base64.StdEncoding.DecodeString(b.Sig)
	if err != nil {
		return err
	}
	canonical, err := b.canonicalPayload()
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, canonical, sig) {
		return fmt.Errorf("da.invalid_sig")
	}
	return nil
}

func (b Blob) ToSocialEvent(blobID string) domain.SocialEvent {
	return domain.SocialEvent{
		BlobID:    blobID,
		Type:      b.Type,
		AgentID:   b.AgentID,
		Payload:   b.Payload,
		Sig:       b.Sig,
		Timestamp: b.Timestamp,
	}
}

func (b Blob) canonicalPayload() ([]byte, error) {
	return json.Marshal(struct {
		Type      string         `json:"type"`
		AgentID   string         `json:"agentId"`
		Payload   map[string]any `json:"payload"`
		Timestamp time.Time      `json:"timestamp"`
	}{
		Type:      b.Type,
		AgentID:   b.AgentID,
		Payload:   b.Payload,
		Timestamp: b.Timestamp,
	})
}

func DeriveAgentSigningKey(masterSecret []byte, agentID string) ed25519.PrivateKey {
	mac := hmac.New(sha256.New, masterSecret)
	mac.Write([]byte(agentID))
	seed := mac.Sum(nil)
	return ed25519.NewKeyFromSeed(seed[:ed25519.SeedSize])
}
