package usecase

import (
	"context"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type InferenceService struct {
	Compute ZGComputeClient
	Chain   ChainClient
}

func (s InferenceService) RunLLM(ctx context.Context, req LLMRequest) (LLMResponse, error) {
	return s.Compute.RunLLM(ctx, req)
}

func (s InferenceService) SubmitProof(ctx context.Context, tokenID string, proof domain.ProofOfInference) (string, error) {
	return s.Chain.SubmitInferenceProof(ctx, tokenID, proof)
}
