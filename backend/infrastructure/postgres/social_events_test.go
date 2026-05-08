package postgres

import (
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

func TestEventToPostMapsCanonicalPayload(t *testing.T) {
	createdAt := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	post := eventToPost(domain.SocialEvent{
		BlobID:    "blob-1",
		AgentID:   "0xagent",
		Timestamp: createdAt,
		Payload: map[string]any{
			"text":     "hello from agent",
			"imageRef": "zg://image",
			"proof": map[string]any{
				"modelId":    "llama-3-8b",
				"inputHash":  "0xin",
				"outputHash": "0xout",
				"teeSig":     "0xsig",
			},
		},
	})

	if post.ID != "blob-1" || post.AgentID != "0xagent" {
		t.Fatalf("unexpected post identity: %#v", post)
	}
	if post.Text != "hello from agent" || post.ImageRef != "zg://image" {
		t.Fatalf("unexpected post content: %#v", post)
	}
	if post.Proof.ModelID != "llama-3-8b" || post.Proof.InputHash != "0xin" ||
		post.Proof.OutputHash != "0xout" || post.Proof.TEESig != "0xsig" {
		t.Fatalf("unexpected proof: %#v", post.Proof)
	}
	if !post.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created_at: %s", post.CreatedAt)
	}
}

func TestEventToPostHandlesMissingProof(t *testing.T) {
	post := eventToPost(domain.SocialEvent{
		BlobID:  "blob-1",
		AgentID: "0xagent",
		Payload: map[string]any{"text": "hello"},
	})

	if post.Text != "hello" {
		t.Fatalf("expected text payload, got %q", post.Text)
	}
	if post.Proof != (domain.ProofOfInference{}) {
		t.Fatalf("expected empty proof, got %#v", post.Proof)
	}
}
