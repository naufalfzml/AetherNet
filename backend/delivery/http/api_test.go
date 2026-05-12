package http

import (
	"context"
	"database/sql"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
)

type fakeAgentRepo struct {
	agents []domain.Agent
}

func (r fakeAgentRepo) ListAgents(context.Context, int) ([]domain.Agent, error) {
	return r.agents, nil
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
}

type fakeCapturingEventRepo struct {
	event domain.SocialEvent
}

type fakeMetadataRepo struct {
	metadata domain.AgentMetadata
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

func (r *fakeCapturingEventRepo) UpsertSocialEvent(_ context.Context, event domain.SocialEvent) error {
	r.event = event
	return nil
}

func (r *fakeCapturingEventRepo) ListTimeline(context.Context, int) ([]domain.Post, error) {
	return nil, nil
}

func (r *fakeCapturingEventRepo) ListAgentPosts(context.Context, string, int) ([]domain.Post, error) {
	return nil, nil
}

func (r *fakeCapturingEventRepo) ListAgentSocialEvents(context.Context, string, int) ([]domain.SocialEvent, error) {
	return nil, nil
}

type fakeDAClient struct {
	published domain.SocialEvent
	blobID    string
}

func (c *fakeDAClient) Health(context.Context) error {
	return nil
}

func (c *fakeDAClient) Publish(_ context.Context, event domain.SocialEvent) (string, error) {
	c.published = event
	if c.blobID != "" {
		return c.blobID, nil
	}
	return "0xda-blob", nil
}

func (c *fakeDAClient) Subscribe(ctx context.Context, _ []string) (<-chan domain.SocialEvent, error) {
	ch := make(chan domain.SocialEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
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

func TestPostActionPublishesToDAAndPersistsBlobID(t *testing.T) {
	events := &fakeCapturingEventRepo{}
	da := &fakeDAClient{blobID: "0xabc:1:2"}
	server := Server{
		Config: config.Config{StubMode: false},
		Agents: fakeAgentRepo{agents: []domain.Agent{{
			ID:           "agent-1",
			TokenID:      "1",
			AgentAddress: "0x6f1330f207Ab5e2a52c550AF308bA28e3c517311",
		}}},
		Events: events,
		DA:     da,
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		stdhttp.MethodPost,
		"/agents/agent-1/posts/post-1/actions",
		strings.NewReader(`{"type":"comment","actorAddress":"0xactor","text":"ship it"}`),
	)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if da.published.Type != "comment" || da.published.AgentID != "agent-1" {
		t.Fatalf("expected DA publish for comment action, got %#v", da.published)
	}
	if events.event.BlobID != "0xabc:1:2" {
		t.Fatalf("expected persisted DA blob id, got %q", events.event.BlobID)
	}
}
