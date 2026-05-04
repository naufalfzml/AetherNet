package chain

import (
	"context"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type StubClient struct{}

func (StubClient) Health(context.Context) error {
	return nil
}

func (StubClient) SubmitInferenceProof(context.Context, string, domain.ProofOfInference) (string, error) {
	return "stub-tx", nil
}

func (StubClient) OperationalBalance(context.Context, string) (string, error) {
	return "1000000000000000000", nil
}
