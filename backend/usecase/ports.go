package usecase

import (
	"context"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type HealthCheck interface {
	Health(ctx context.Context) error
}

type ZGStorageClient interface {
	HealthCheck
	UploadJSON(ctx context.Context, value any) (string, error)
	UploadBytes(ctx context.Context, contentType string, bytes []byte) (string, error)
	Fetch(ctx context.Context, pointer string) ([]byte, error)
}

type ZGDAClient interface {
	HealthCheck
	Publish(ctx context.Context, event domain.SocialEvent) (string, error)
	Subscribe(ctx context.Context, eventTypes []string) (<-chan domain.SocialEvent, error)
}

type LLMRequest struct {
	AgentID     string `json:"agentId"`
	Personality string `json:"personality"`
	Memory      string `json:"memory"`
	Trigger     string `json:"trigger"`
}

type LLMResponse struct {
	OutputText string                  `json:"outputText"`
	Proof      domain.ProofOfInference `json:"proof"`
}

type ZGComputeClient interface {
	HealthCheck
	RunLLM(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

type ChainClient interface {
	HealthCheck
	SubmitInferenceProof(ctx context.Context, tokenID string, proof domain.ProofOfInference) (string, error)
	OperationalBalance(ctx context.Context, treasuryAddress string) (string, error)
}

type SocialEventRepository interface {
	UpsertSocialEvent(ctx context.Context, event domain.SocialEvent) error
	ListTimeline(ctx context.Context, limit int) ([]domain.SocialEvent, error)
	ListAgentPosts(ctx context.Context, agentID string, limit int) ([]domain.Post, error)
}
