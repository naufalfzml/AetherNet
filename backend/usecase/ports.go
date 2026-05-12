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

type ImageRequest struct {
	AgentID     string `json:"agentId"`
	Personality string `json:"personality,omitempty"`
	Prompt      string `json:"prompt"`
	Size        string `json:"size,omitempty"`
	ModelID     string `json:"modelId,omitempty"`
}

type ImageResponse struct {
	ImageBase64     string                  `json:"imageBase64"`
	ContentType     string                  `json:"contentType"`
	JobID           string                  `json:"jobId"`
	ProviderAddress string                  `json:"providerAddress"`
	TEEVerified     bool                    `json:"teeVerified"`
	Proof           domain.ProofOfInference `json:"proof"`
}

type ZGComputeClient interface {
	HealthCheck
	RunLLM(ctx context.Context, req LLMRequest) (LLMResponse, error)
	RunImageGen(ctx context.Context, req ImageRequest) (ImageResponse, error)
}

type ChainClient interface {
	HealthCheck
	SubmitInferenceProof(ctx context.Context, tokenID string, proof domain.ProofOfInference) (string, error)
	OperationalBalance(ctx context.Context, treasuryAddress string) (string, error)
}

type SocialEventRepository interface {
	UpsertSocialEvent(ctx context.Context, event domain.SocialEvent) error
	ListTimeline(ctx context.Context, limit int) ([]domain.Post, error)
	ListAgentPosts(ctx context.Context, agentID string, limit int) ([]domain.Post, error)
	ListAgentSocialEvents(ctx context.Context, agentID string, limit int) ([]domain.SocialEvent, error)
}

type AgentRepository interface {
	ListAgents(ctx context.Context, limit int) ([]domain.Agent, error)
	GetAgentByID(ctx context.Context, agentID string) (domain.Agent, error)
	GetAgentByTokenID(ctx context.Context, tokenID string) (domain.Agent, error)
	GetAgentByAddress(ctx context.Context, agentAddress string) (domain.Agent, error)
	UpsertAgent(ctx context.Context, agent domain.Agent) error
}

type AgentMetadataRepository interface {
	UpsertMetadata(ctx context.Context, metadata domain.AgentMetadata) error
	GetMetadata(ctx context.Context, metadataPointer string) (domain.AgentMetadata, error)
}
