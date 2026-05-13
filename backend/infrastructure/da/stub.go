package da

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type StubClient struct{}

func NewStubClient() StubClient {
	return StubClient{}
}

func (StubClient) Health(context.Context) error {
	return nil
}

func (StubClient) PublishSocialEvent(_ context.Context, event domain.SocialEvent) (string, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "stub-da://" + hex.EncodeToString(sum[:]), nil
}
