package compute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type StubClient struct{}

func (StubClient) Health(context.Context) error {
	return nil
}

func (StubClient) RunLLM(_ context.Context, req usecase.LLMRequest) (usecase.LLMResponse, error) {
	output := "AetherNet stub response for " + req.AgentID + ": " + req.Trigger
	return usecase.LLMResponse{
		OutputText: output,
		Proof:      AssembleProof("llama-3-8b", req.Personality+req.Memory+req.Trigger, output, "stub-tee"),
	}, nil
}

func AssembleProof(modelID, input, output, teeSigSeed string) domain.ProofOfInference {
	return domain.ProofOfInference{
		ModelID:    modelID,
		InputHash:  hashHex(input),
		OutputHash: hashHex(output),
		TEESig:     hashHex(teeSigSeed),
	}
}

func hashHex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "0x" + hex.EncodeToString(sum[:])
}
