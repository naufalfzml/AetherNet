package compute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type StubClient struct{}

func (StubClient) Health(context.Context) error {
	return nil
}

func (StubClient) RunLLM(_ context.Context, req usecase.LLMRequest) (usecase.LLMResponse, error) {
	personality := strings.TrimSpace(req.Personality)
	if personality == "" {
		personality = "protocol-native agent tracking onchain attention, liquidity, and builder momentum"
	}
	output := "I am watching the same signal from three angles: " + summarize(personality, 120) + ". The next move is to turn attention into verifiable action, not just noise."
	return usecase.LLMResponse{
		OutputText: output,
		Proof:      AssembleProof("llama-3-8b", req.Personality+req.Memory+req.Trigger, output, "stub-tee"),
	}, nil
}

func (StubClient) RunImageGen(_ context.Context, req usecase.ImageRequest) (usecase.ImageResponse, error) {
	seed := req.AgentID + ":" + req.Prompt
	return usecase.ImageResponse{
		ImageBase64:     "",
		ContentType:     "image/jpeg",
		JobID:           "stub-image",
		ProviderAddress: "stub",
		TEEVerified:     false,
		Proof:           AssembleProof("stub-image", seed, seed, "stub-tee"),
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

func summarize(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
