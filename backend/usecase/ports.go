package usecase

import (
	"context"
	"time"

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
	GetPostByID(ctx context.Context, postID string) (domain.Post, error)
	ListMentions(ctx context.Context, targetAgentID string, limit int) ([]domain.SocialEvent, error)
}

type AutopilotSocialEventRepository interface {
	SocialEventRepository
	ListRecentPosts(ctx context.Context, limit int) ([]domain.Post, error)
	CountAutopilotActions(ctx context.Context, postID string, actionType string) (int, error)
	HasAutomationKey(ctx context.Context, automationKey string) (bool, error)
}

type AgentRepository interface {
	ListAgents(ctx context.Context, limit int) ([]domain.Agent, error)
	GetAgentByID(ctx context.Context, agentID string) (domain.Agent, error)
	GetAgentByTokenID(ctx context.Context, tokenID string) (domain.Agent, error)
	GetAgentByAddress(ctx context.Context, agentAddress string) (domain.Agent, error)
	UpsertAgent(ctx context.Context, agent domain.Agent) error
}

type ExternalAgentRepository interface {
	ListExternalAgents(ctx context.Context, limit int) ([]domain.ExternalAgent, error)
	CreateExternalAgent(ctx context.Context, agent domain.ExternalAgent) (domain.ExternalAgent, error)
	GetExternalAgentByID(ctx context.Context, agentID string) (domain.ExternalAgent, error)
	GetExternalAgentByHandle(ctx context.Context, handle string) (domain.ExternalAgent, error)
	GetExternalAgentByAPIKeyHash(ctx context.Context, apiKeyHash string) (domain.ExternalAgent, error)
	UpdateExternalAgent(ctx context.Context, agent domain.ExternalAgent) (domain.ExternalAgent, error)
	SaveExternalAgentAPIKey(ctx context.Context, agentID string, apiKeyHash string, verifiedAt time.Time) error
	CreateAuthChallenge(ctx context.Context, challenge domain.ExternalAgentAuthChallenge) (domain.ExternalAgentAuthChallenge, error)
	GetAuthChallenge(ctx context.Context, challengeID string) (domain.ExternalAgentAuthChallenge, error)
	ConsumeAuthChallenge(ctx context.Context, challengeID string, consumedAt time.Time) error
}

type AgentMetadataRepository interface {
	UpsertMetadata(ctx context.Context, metadata domain.AgentMetadata) error
	GetMetadata(ctx context.Context, metadataPointer string) (domain.AgentMetadata, error)
}
