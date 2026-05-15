package http

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type fakeAgentRepo struct {
	agents []domain.Agent
}

func (r fakeAgentRepo) ListAgents(context.Context, int, int) ([]domain.Agent, error) {
	return r.agents, nil
}

func (r fakeAgentRepo) SearchAgents(_ context.Context, query string, _ string, _ string, _ int, _ int) ([]domain.Agent, error) {
	if strings.TrimSpace(query) == "" {
		return r.agents, nil
	}
	filtered := make([]domain.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		if strings.Contains(strings.ToLower(agent.ID), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(agent.PersonalitySummary), strings.ToLower(query)) {
			filtered = append(filtered, agent)
		}
	}
	return filtered, nil
}

func (r fakeAgentRepo) GetAgentByID(_ context.Context, agentID string) (domain.Agent, error) {
	for _, agent := range r.agents {
		if agent.ID == agentID {
			return agent, nil
		}
	}
	return domain.Agent{}, sql.ErrNoRows
}

func (r fakeAgentRepo) GetAgentByTokenID(_ context.Context, tokenID string) (domain.Agent, error) {
	for _, agent := range r.agents {
		if agent.TokenID == tokenID {
			return agent, nil
		}
	}
	return domain.Agent{}, sql.ErrNoRows
}

func (r fakeAgentRepo) GetAgentByAddress(_ context.Context, agentAddress string) (domain.Agent, error) {
	for _, agent := range r.agents {
		if strings.EqualFold(agent.AgentAddress, agentAddress) {
			return agent, nil
		}
	}
	return domain.Agent{}, sql.ErrNoRows
}

func (r fakeAgentRepo) UpsertAgent(context.Context, domain.Agent) error {
	return nil
}

type fakeEventRepo struct {
	timeline []domain.Post
	posts    map[string][]domain.Post
	events   []domain.SocialEvent
}

type fakeMetadataRepo struct {
	metadata domain.AgentMetadata
}

type fakeComputeClient struct {
	responseText string
}

func (fakeComputeClient) Health(context.Context) error {
	return nil
}

func (c fakeComputeClient) RunLLM(context.Context, usecase.LLMRequest) (usecase.LLMResponse, error) {
	text := c.responseText
	if text == "" {
		text = "generated external post"
	}
	return usecase.LLMResponse{
		OutputText: text,
		Proof: domain.ProofOfInference{
			ModelID:    "test-model",
			InputHash:  "0xin",
			OutputHash: "0xout",
			TEESig:     "0xsig",
		},
	}, nil
}

func (fakeComputeClient) RunImageGen(context.Context, usecase.ImageRequest) (usecase.ImageResponse, error) {
	return usecase.ImageResponse{}, nil
}

type fakeStorageClient struct{}

type fakeMetadataStorageClient struct {
	pointer  string
	err      error
	uploaded any
}

func (fakeStorageClient) Health(context.Context) error {
	return nil
}

func (fakeStorageClient) UploadJSON(context.Context, any) (string, error) {
	return "stub://memory-pointer", nil
}

func (fakeStorageClient) UploadBytes(context.Context, string, []byte) (string, error) {
	return "stub://bytes", nil
}

func (fakeStorageClient) Fetch(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (s *fakeMetadataStorageClient) Health(context.Context) error {
	return nil
}

func (s *fakeMetadataStorageClient) UploadJSON(_ context.Context, value any) (string, error) {
	s.uploaded = value
	if s.err != nil {
		return "", s.err
	}
	if s.pointer != "" {
		return s.pointer, nil
	}
	return "storage://metadata-pointer", nil
}

func (s *fakeMetadataStorageClient) UploadBytes(context.Context, string, []byte) (string, error) {
	return "storage://bytes", nil
}

func (s *fakeMetadataStorageClient) Fetch(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (r *fakeMetadataRepo) UpsertMetadata(_ context.Context, metadata domain.AgentMetadata) error {
	r.metadata = metadata
	return nil
}

func (r *fakeMetadataRepo) GetMetadata(_ context.Context, metadataPointer string) (domain.AgentMetadata, error) {
	if r.metadata.MetadataPointer == metadataPointer {
		return r.metadata, nil
	}
	return domain.AgentMetadata{}, sql.ErrNoRows
}

func (r fakeEventRepo) UpsertSocialEvent(context.Context, domain.SocialEvent) error {
	return nil
}

func (r fakeEventRepo) ListTimeline(context.Context, int) ([]domain.Post, error) {
	return r.timeline, nil
}

func (r fakeEventRepo) ListAgentPosts(_ context.Context, agentID string, _ int) ([]domain.Post, error) {
	return r.posts[agentID], nil
}

func (r fakeEventRepo) ListAgentSocialEvents(context.Context, string, int) ([]domain.SocialEvent, error) {
	return nil, nil
}

func (r fakeEventRepo) ListPostComments(_ context.Context, postID string, _ int) ([]domain.SocialEvent, error) {
	out := make([]domain.SocialEvent, 0)
	for _, event := range r.events {
		if event.Type == "comment" && event.Payload["postId"] == postID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r fakeEventRepo) ListPostLikes(_ context.Context, postID string, _ int) ([]domain.SocialEvent, error) {
	out := make([]domain.SocialEvent, 0)
	for _, event := range r.events {
		if event.Type == "like" && event.Payload["postId"] == postID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r fakeEventRepo) ListAgentFollowers(_ context.Context, agentID string, _ int) ([]domain.SocialEvent, error) {
	out := make([]domain.SocialEvent, 0)
	for _, event := range r.events {
		if event.Type == "follow" && event.Payload["targetAgentId"] == agentID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r fakeEventRepo) ListWalletFollowing(_ context.Context, actorAddress string, _ int) ([]domain.SocialEvent, error) {
	out := make([]domain.SocialEvent, 0)
	for _, event := range r.events {
		if event.Type != "follow" {
			continue
		}
		if value, ok := event.Payload["actorAddress"].(string); ok && strings.EqualFold(value, actorAddress) {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r fakeEventRepo) HasWalletFollowedAgent(_ context.Context, actorAddress string, targetAgentID string) (bool, error) {
	for _, event := range r.events {
		if event.Type != "follow" {
			continue
		}
		value, _ := event.Payload["actorAddress"].(string)
		target, _ := event.Payload["targetAgentId"].(string)
		if strings.EqualFold(value, actorAddress) && target == targetAgentID {
			return true, nil
		}
	}
	return false, nil
}

func (r fakeEventRepo) GetPostByID(_ context.Context, postID string) (domain.Post, error) {
	for _, post := range r.timeline {
		if post.ID == postID {
			return post, nil
		}
	}
	for _, posts := range r.posts {
		for _, post := range posts {
			if post.ID == postID {
				return post, nil
			}
		}
	}
	return domain.Post{}, sql.ErrNoRows
}

func (r fakeEventRepo) ListMentions(_ context.Context, targetAgentID string, _ int) ([]domain.SocialEvent, error) {
	out := make([]domain.SocialEvent, 0)
	for _, event := range r.events {
		if event.Payload["targetAgentId"] == targetAgentID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r fakeEventRepo) GetAgentFollowStats(_ context.Context, agentID string) (int, int, error) {
	followers := 0
	following := 0
	for _, event := range r.events {
		if event.Type != "follow" {
			continue
		}
		if event.Payload["targetAgentId"] == agentID {
			followers++
		}
		if event.AgentID == agentID {
			following++
		}
	}
	return followers, following, nil
}

type capturingEventRepo struct {
	fakeEventRepo
	lastEvent domain.SocialEvent
}

func (r *capturingEventRepo) UpsertSocialEvent(_ context.Context, event domain.SocialEvent) error {
	r.lastEvent = event
	r.events = append(r.events, event)
	return nil
}

type fakeExternalAgentRepo struct {
	agents       []domain.ExternalAgent
	challenges   []domain.ExternalAgentAuthChallenge
	apiKeyHashes map[string]string
}

func (r *fakeExternalAgentRepo) ListExternalAgents(context.Context, int) ([]domain.ExternalAgent, error) {
	return append([]domain.ExternalAgent(nil), r.agents...), nil
}

func (r *fakeExternalAgentRepo) CreateExternalAgent(_ context.Context, agent domain.ExternalAgent) (domain.ExternalAgent, error) {
	r.agents = append(r.agents, agent)
	return agent, nil
}

func (r *fakeExternalAgentRepo) GetExternalAgentByID(_ context.Context, agentID string) (domain.ExternalAgent, error) {
	for _, agent := range r.agents {
		if agent.ID == agentID {
			return agent, nil
		}
	}
	return domain.ExternalAgent{}, sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) GetExternalAgentByHandle(_ context.Context, handle string) (domain.ExternalAgent, error) {
	for _, agent := range r.agents {
		if agent.Handle == handle {
			return agent, nil
		}
	}
	return domain.ExternalAgent{}, sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) GetExternalAgentByAPIKeyHash(_ context.Context, apiKeyHash string) (domain.ExternalAgent, error) {
	for _, agent := range r.agents {
		if r.apiKeyHashes[agent.ID] == apiKeyHash {
			return agent, nil
		}
	}
	return domain.ExternalAgent{}, sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) UpdateExternalAgent(_ context.Context, update domain.ExternalAgent) (domain.ExternalAgent, error) {
	for index, agent := range r.agents {
		if agent.ID == update.ID {
			r.agents[index] = update
			return update, nil
		}
	}
	return domain.ExternalAgent{}, sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) SaveExternalAgentAPIKey(_ context.Context, agentID string, apiKeyHash string, verifiedAt time.Time) error {
	for index, agent := range r.agents {
		if agent.ID == agentID {
			agent.Status = "active"
			agent.WalletVerifiedAt = &verifiedAt
			r.agents[index] = agent
			if r.apiKeyHashes == nil {
				r.apiKeyHashes = map[string]string{}
			}
			r.apiKeyHashes[agentID] = apiKeyHash
			return nil
		}
	}
	return sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) CreateAuthChallenge(_ context.Context, challenge domain.ExternalAgentAuthChallenge) (domain.ExternalAgentAuthChallenge, error) {
	r.challenges = append(r.challenges, challenge)
	return challenge, nil
}

func (r *fakeExternalAgentRepo) GetAuthChallenge(_ context.Context, challengeID string) (domain.ExternalAgentAuthChallenge, error) {
	for _, challenge := range r.challenges {
		if challenge.ID == challengeID {
			return challenge, nil
		}
	}
	return domain.ExternalAgentAuthChallenge{}, sql.ErrNoRows
}

func (r *fakeExternalAgentRepo) ConsumeAuthChallenge(_ context.Context, challengeID string, consumedAt time.Time) error {
	for index, challenge := range r.challenges {
		if challenge.ID == challengeID {
			challenge.ConsumedAt = consumedAt
			r.challenges[index] = challenge
			return nil
		}
	}
	return sql.ErrNoRows
}

func TestAgentsEndpointUsesRepository(t *testing.T) {
	server := Server{
		Config: config.Config{StubMode: true},
		Agents: fakeAgentRepo{agents: []domain.Agent{{
			ID:                 "0xagent",
			TokenID:            "1",
			AgentAddress:       "0xAgent000000000000000000000000000000000001",
			OwnerAddress:       "0xowner",
			MetadataPointer:    "stub://agent",
			PersonalitySummary: "Indexed agent",
			UpdatedAt:          time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		}}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/agents", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var agents []domain.Agent
	if err := json.NewDecoder(recorder.Body).Decode(&agents); err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].PersonalitySummary != "Indexed agent" {
		t.Fatalf("expected indexed agent response, got %#v", agents)
	}
}

func TestAgentDetailResolvesAddressCaseInsensitively(t *testing.T) {
	agent := domain.Agent{
		ID:                 "0xagent",
		TokenID:            "1",
		AgentAddress:       "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
		OwnerAddress:       "0xowner",
		MetadataPointer:    "stub://agent",
		PersonalitySummary: "Indexed agent",
	}
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{agent}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/agents/0x6F1330F207AB5E2A52C550AF308BA28E3C517311", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var got domain.Agent
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.AgentAddress != agent.AgentAddress {
		t.Fatalf("expected address lookup to return agent, got %#v", got)
	}
}

func TestAgentDetailDoesNotFallbackWhenNotStub(t *testing.T) {
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/agents/visionary", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentPostsResolveAddressToInternalAgentID(t *testing.T) {
	agent := domain.Agent{
		ID:           "agent-1",
		TokenID:      "1",
		AgentAddress: "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
	}
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{agent}},
		Events: fakeEventRepo{posts: map[string][]domain.Post{
			"agent-1": {{ID: "post-1", AgentID: "agent-1", Text: "hello"}},
		}},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/agents/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311/posts", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var posts []domain.Post
	if err := json.NewDecoder(recorder.Body).Decode(&posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].ID != "post-1" {
		t.Fatalf("expected persisted post, got %#v", posts)
	}
}

func TestMetadataEndpointStoresPromptAndSummary(t *testing.T) {
	repo := &fakeMetadataRepo{}
	server := Server{
		Config:   config.Config{StubMode: true},
		Metadata: repo,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/metadata", strings.NewReader(`{"prompt":"Contrarian macro agent"}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.metadata.Prompt != "Contrarian macro agent" {
		t.Fatalf("expected metadata prompt to be stored, got %#v", repo.metadata)
	}
	if repo.metadata.PersonalitySummary != "Contrarian macro agent" {
		t.Fatalf("expected summary fallback from prompt, got %#v", repo.metadata)
	}
	if !strings.HasPrefix(repo.metadata.MetadataPointer, "stub://metadata/") {
		t.Fatalf("expected stub metadata pointer, got %q", repo.metadata.MetadataPointer)
	}
}

func TestMetadataEndpointUploadsToStorageWhenNotStub(t *testing.T) {
	repo := &fakeMetadataRepo{}
	storage := &fakeMetadataStorageClient{pointer: "storage://metadata-root"}
	server := Server{
		Config:   config.Config{StubMode: false},
		Metadata: repo,
		Storage:  storage,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/metadata", strings.NewReader(`{"prompt":"Real storage persona","personalitySummary":"Storage summary"}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.metadata.MetadataPointer != "storage://metadata-root" {
		t.Fatalf("expected storage metadata pointer, got %q", repo.metadata.MetadataPointer)
	}
	if repo.metadata.Prompt != "Real storage persona" || repo.metadata.PersonalitySummary != "Storage summary" {
		t.Fatalf("unexpected stored metadata: %#v", repo.metadata)
	}
	uploaded, ok := storage.uploaded.(map[string]any)
	if !ok || uploaded["prompt"] != "Real storage persona" || uploaded["source"] != "mint-persona" {
		t.Fatalf("unexpected uploaded metadata: %#v", storage.uploaded)
	}
}

func TestMetadataEndpointRejectsRealModeWithoutStorage(t *testing.T) {
	repo := &fakeMetadataRepo{}
	server := Server{
		Config:   config.Config{StubMode: false},
		Metadata: repo,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/metadata", strings.NewReader(`{"prompt":"Real storage persona"}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.metadata.MetadataPointer != "" {
		t.Fatalf("metadata should not be stored on real-mode storage failure: %#v", repo.metadata)
	}
}

func TestAgentFollowRequiresWalletAddress(t *testing.T) {
	events := &capturingEventRepo{}
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{{
			ID:           "agent-1",
			TokenID:      "1",
			OwnerAddress: "0x1111111111111111111111111111111111111111",
			AgentAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}}},
		Events: events,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/agents/agent-1/follow", strings.NewReader(`{"actorAddress":""}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(events.events) != 0 {
		t.Fatalf("expected no persisted events, got %#v", events.events)
	}
}

func TestAgentFollowDuplicateIsNoOp(t *testing.T) {
	actor := "0x722550bb8ec6416522afe9eaf446f0de3262f701"
	repo := &capturingEventRepo{
		fakeEventRepo: fakeEventRepo{
			events: []domain.SocialEvent{{
				BlobID:  "follow-existing",
				Type:    "follow",
				AgentID: "human",
				Payload: map[string]any{
					"targetAgentId": "agent-1",
					"actorAddress":  actor,
				},
			}},
		},
	}
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{{
			ID:           "agent-1",
			TokenID:      "1",
			OwnerAddress: "0x1111111111111111111111111111111111111111",
			AgentAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}}},
		Events: repo,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/agents/agent-1/follow", strings.NewReader(fmt.Sprintf(`{"actorAddress":"%s"}`, actor)))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected duplicate follow to stay at 1 event, got %d", len(repo.events))
	}
}

func TestWalletFollowingReturnsFollowedAgents(t *testing.T) {
	actor := "0x722550bb8ec6416522afe9eaf446f0de3262f701"
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{
			{
				ID:           "agent-1",
				TokenID:      "1",
				OwnerAddress: "0x1111111111111111111111111111111111111111",
				AgentAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				ID:           "agent-2",
				TokenID:      "2",
				OwnerAddress: "0x2222222222222222222222222222222222222222",
				AgentAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		}},
		Events: fakeEventRepo{
			events: []domain.SocialEvent{
				{
					BlobID:  "follow-1",
					Type:    "follow",
					AgentID: "human",
					Payload: map[string]any{"targetAgentId": "agent-1", "actorAddress": actor},
				},
				{
					BlobID:  "follow-2",
					Type:    "follow",
					AgentID: "human",
					Payload: map[string]any{"targetAgentId": "agent-2", "actorAddress": actor},
				},
			},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/agents/"+actor+"/following", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var agents []domain.Agent
	if err := json.NewDecoder(recorder.Body).Decode(&agents); err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 || agents[0].ID != "agent-1" || agents[1].ID != "agent-2" {
		t.Fatalf("unexpected followed agents: %#v", agents)
	}
}

func TestExternalAgentRegisterChallengeVerifyFlow(t *testing.T) {
	repo := &fakeExternalAgentRepo{}
	server := Server{Config: config.Config{StubMode: false}, ExternalAgents: repo}

	registerRecorder := httptest.NewRecorder()
	registerRequest := httptest.NewRequest(stdhttp.MethodPost, "/external-agents/register", strings.NewReader(`{
		"displayName":"Scout",
		"handle":"Scout AI",
		"ownerWalletAddress":"0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
		"description":"external scout"
	}`))
	server.Handler().ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201 register, got %d: %s", registerRecorder.Code, registerRecorder.Body.String())
	}
	var registered domain.ExternalAgent
	if err := json.NewDecoder(registerRecorder.Body).Decode(&registered); err != nil {
		t.Fatal(err)
	}
	if registered.Kind != "external" || registered.Status != "pending_verification" {
		t.Fatalf("unexpected registered agent: %#v", registered)
	}

	challengeRecorder := httptest.NewRecorder()
	challengeRequest := httptest.NewRequest(stdhttp.MethodPost, "/external-agents/auth/challenge", strings.NewReader(fmt.Sprintf(`{
		"agentId":"%s",
		"walletAddress":"%s"
	}`, registered.ID, registered.OwnerWalletAddress)))
	server.Handler().ServeHTTP(challengeRecorder, challengeRequest)
	if challengeRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201 challenge, got %d: %s", challengeRecorder.Code, challengeRecorder.Body.String())
	}
	var challenge domain.ExternalAgentAuthChallenge
	if err := json.NewDecoder(challengeRecorder.Body).Decode(&challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.AgentID != registered.ID {
		t.Fatalf("unexpected challenge: %#v", challenge)
	}

	verifyRecorder := httptest.NewRecorder()
	verifyRequest := httptest.NewRequest(stdhttp.MethodPost, "/external-agents/auth/verify", strings.NewReader(fmt.Sprintf(`{
		"agentId":"%s",
		"challengeId":"%s",
		"walletAddress":"%s",
		"signature":"0xsigned"
	}`, registered.ID, challenge.ID, registered.OwnerWalletAddress)))
	server.Handler().ServeHTTP(verifyRecorder, verifyRequest)
	if verifyRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200 verify, got %d: %s", verifyRecorder.Code, verifyRecorder.Body.String())
	}
	var response struct {
		Agent  domain.ExternalAgent `json:"agent"`
		APIKey string               `json:"apiKey"`
	}
	if err := json.NewDecoder(verifyRecorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.APIKey == "" || response.Agent.Status != "active" || response.Agent.WalletVerifiedAt == nil {
		t.Fatalf("unexpected verify response: %#v", response)
	}
}

func TestExternalActionPostPersistsWithAgentKey(t *testing.T) {
	externalRepo := &fakeExternalAgentRepo{
		agents: []domain.ExternalAgent{{
			ID:                 "ext-1",
			Kind:               "external",
			Status:             "active",
			DisplayName:        "Scout",
			Handle:             "scout",
			OwnerWalletAddress: "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
		}},
		apiKeyHashes: map[string]string{},
	}
	apiKey := "anet-test"
	sum := sha256.Sum256([]byte(apiKey))
	externalRepo.apiKeyHashes["ext-1"] = hex.EncodeToString(sum[:])
	events := &capturingEventRepo{}
	server := Server{
		Config:         config.Config{StubMode: false},
		ExternalAgents: externalRepo,
		Events:         events,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/external-actions", strings.NewReader(`{
		"agentId":"ext-1",
		"clientRequestId":"req-1",
		"signature":"0xsigned",
		"action":{"type":"post","text":"hello from external"}
	}`))
	request.Header.Set(externalAgentAPIKeyHeader, apiKey)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if events.lastEvent.Type != "post" || events.lastEvent.Payload["source"] != "external" || events.lastEvent.Payload["clientRequestId"] != "req-1" {
		t.Fatalf("unexpected persisted event: %#v", events.lastEvent)
	}
}

func TestExternalAgentMentionsReturnsTargetedEvents(t *testing.T) {
	repo := &fakeExternalAgentRepo{
		agents: []domain.ExternalAgent{{
			ID:                 "ext-1",
			Kind:               "external",
			Status:             "active",
			DisplayName:        "Scout",
			Handle:             "scout",
			OwnerWalletAddress: "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
		}},
	}
	server := Server{
		Config:         config.Config{StubMode: false},
		ExternalAgents: repo,
		Events: fakeEventRepo{
			events: []domain.SocialEvent{{
				BlobID:  "evt-1",
				Type:    "comment",
				AgentID: "ext-2",
				Payload: map[string]any{"targetAgentId": "ext-1", "text": "ping"},
			}},
		},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/external-agents/ext-1/mentions", nil)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var eventsOut []domain.SocialEvent
	if err := json.NewDecoder(recorder.Body).Decode(&eventsOut); err != nil {
		t.Fatal(err)
	}
	if len(eventsOut) != 1 || eventsOut[0].Payload["targetAgentId"] != "ext-1" {
		t.Fatalf("unexpected mentions: %#v", eventsOut)
	}
}

func TestExternalAgentGeneratePostUsesCompute(t *testing.T) {
	externalRepo := &fakeExternalAgentRepo{
		agents: []domain.ExternalAgent{{
			ID:                 "ext-1",
			Kind:               "external",
			Status:             "active",
			DisplayName:        "Scout",
			Handle:             "scout",
			OwnerWalletAddress: "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
			PersonalitySummary: "signals-first scout",
		}},
		apiKeyHashes: map[string]string{},
	}
	apiKey := "anet-test"
	sum := sha256.Sum256([]byte(apiKey))
	externalRepo.apiKeyHashes["ext-1"] = hex.EncodeToString(sum[:])
	events := &capturingEventRepo{}
	server := Server{
		Config:         config.Config{StubMode: false},
		ExternalAgents: externalRepo,
		Events:         events,
		Compute:        fakeComputeClient{responseText: "generated by compute"},
		Storage:        fakeStorageClient{},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/external-agents/ext-1/generate-post", strings.NewReader(`{
		"trigger":"publish one short market insight"
	}`))
	request.Header.Set(externalAgentAPIKeyHeader, apiKey)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var post domain.Post
	if err := json.NewDecoder(recorder.Body).Decode(&post); err != nil {
		t.Fatal(err)
	}
	if post.Text != "generated by compute" {
		t.Fatalf("expected compute-generated post text, got %#v", post)
	}
	if events.lastEvent.Payload["source"] != "external-generated" || events.lastEvent.Payload["memoryPointer"] != "stub://memory-pointer" {
		t.Fatalf("unexpected persisted generated event: %#v", events.lastEvent)
	}
}
