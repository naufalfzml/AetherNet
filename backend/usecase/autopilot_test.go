package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

func TestAutopilotSkipsSelfActionsAndDuplicates(t *testing.T) {
	ctx := context.Background()
	events := newFakeAutopilotEvents([]domain.Post{{ID: "post-1", AgentID: "agent-a", Text: "hello"}})
	events.existing[AutomationKey("agent-b", "like", "post-1")] = true
	events.existing[AutomationKey("agent-c", "comment", "post-1")] = true
	ap := testAutopilot(events, []domain.Agent{
		{ID: "agent-a", PersonalitySummary: "author"},
		{ID: "agent-b", PersonalitySummary: "builder"},
		{ID: "agent-c", PersonalitySummary: "critic"},
	})

	result, err := ap.Tick(ctx)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	if result.LikesCreated != 1 || result.CommentsCreated != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	for _, event := range events.persisted {
		if event.AgentID == "agent-a" {
			t.Fatalf("self action persisted: %#v", event)
		}
		if event.Payload["automationKey"] == AutomationKey("agent-b", "like", "post-1") {
			t.Fatalf("duplicate like persisted: %#v", event)
		}
		if event.Payload["automationKey"] == AutomationKey("agent-c", "comment", "post-1") {
			t.Fatalf("duplicate comment persisted: %#v", event)
		}
	}
}

func TestAutopilotDefaultAndOverrideCaps(t *testing.T) {
	ctx := context.Background()
	posts := []domain.Post{
		{ID: "post-1", AgentID: "agent-a", Text: "one"},
		{ID: "post-2", AgentID: "agent-a", Text: "two"},
		{ID: "post-3", AgentID: "agent-a", Text: "three"},
	}
	agents := []domain.Agent{
		{ID: "agent-a"},
		{ID: "agent-b"},
		{ID: "agent-c"},
		{ID: "agent-d"},
		{ID: "agent-e"},
	}

	defaultEvents := newFakeAutopilotEvents(posts)
	defaultAP := testAutopilot(defaultEvents, agents)
	defaultResult, err := defaultAP.Tick(ctx)
	if err != nil {
		t.Fatalf("default tick failed: %v", err)
	}
	if defaultResult.PostsProcessed != 3 || defaultResult.LikesCreated != 9 || defaultResult.CommentsCreated != 6 {
		t.Fatalf("default caps not enforced: %#v", defaultResult)
	}

	overrideEvents := newFakeAutopilotEvents(posts)
	overrideAP := testAutopilot(overrideEvents, agents)
	overrideAP.Config = AutopilotConfig{MaxPostsPerTick: 1, MaxLikesPerPost: 1, MaxCommentsPerPost: 1}
	overrideResult, err := overrideAP.Tick(ctx)
	if err != nil {
		t.Fatalf("override tick failed: %v", err)
	}
	if overrideResult.PostsProcessed != 1 || overrideResult.LikesCreated != 1 || overrideResult.CommentsCreated != 1 {
		t.Fatalf("override caps not enforced: %#v", overrideResult)
	}
}

func TestAutopilotPersistsHybridEventsWithoutDA(t *testing.T) {
	events := newFakeAutopilotEvents([]domain.Post{{ID: "post-1", AgentID: "agent-a", Text: "hello"}})
	ap := testAutopilot(events, []domain.Agent{{ID: "agent-a"}, {ID: "agent-b"}})
	ap.Config = AutopilotConfig{MaxLikesPerPost: 1, MaxCommentsPerPost: 1}

	result, err := ap.Tick(context.Background())
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	if result.LikesCreated != 1 || result.CommentsCreated != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(events.persisted) != 2 {
		t.Fatalf("expected persisted like and comment, got %#v", events.persisted)
	}
	for _, event := range events.persisted {
		if !strings.HasPrefix(event.BlobID, "hybrid-autopilot-"+event.Type+"-") {
			t.Fatalf("expected hybrid autopilot id, got %#v", event)
		}
		if strings.HasPrefix(event.BlobID, "stub-da://") {
			t.Fatalf("unexpected DA blob id: %#v", event)
		}
	}
}

func TestAutopilotComputeFailurePreventsCommentPublishAndPersistence(t *testing.T) {
	events := newFakeAutopilotEvents([]domain.Post{{ID: "post-1", AgentID: "agent-a", Text: "hello"}})
	ap := testAutopilot(events, []domain.Agent{{ID: "agent-a"}, {ID: "agent-b"}})
	ap.Config = AutopilotConfig{MaxLikesPerPost: 0, MaxCommentsPerPost: 1}
	ap.Compute = fakeCompute{err: errors.New("compute down")}

	_, err := ap.Tick(context.Background())
	if err == nil {
		t.Fatalf("expected compute error")
	}
	for _, event := range events.persisted {
		if event.Type == "comment" {
			t.Fatalf("comment persisted after compute failure: %#v", event)
		}
	}
}

func TestAutopilotMemorySuccessAndFailurePayloads(t *testing.T) {
	ctx := context.Background()
	events := newFakeAutopilotEvents([]domain.Post{{ID: "post-1", AgentID: "agent-a", Text: "hello"}})
	ap := testAutopilot(events, []domain.Agent{{ID: "agent-a"}, {ID: "agent-b"}})
	ap.Config = AutopilotConfig{MaxLikesPerPost: 0, MaxCommentsPerPost: 1}
	if _, err := ap.Tick(ctx); err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	comment := events.firstType("comment")
	if comment.Payload["memoryStatus"] != "updated" || comment.Payload["memoryPointer"] == "" {
		t.Fatalf("missing successful memory payload: %#v", comment.Payload)
	}

	failedEvents := newFakeAutopilotEvents([]domain.Post{{ID: "post-1", AgentID: "agent-a", Text: "hello"}})
	failedAP := testAutopilot(failedEvents, []domain.Agent{{ID: "agent-a"}, {ID: "agent-b"}})
	failedAP.Config = AutopilotConfig{MaxLikesPerPost: 0, MaxCommentsPerPost: 1}
	failedAP.Storage = fakeStorage{err: errors.New("storage down")}
	if _, err := failedAP.Tick(ctx); err != nil {
		t.Fatalf("tick with failed storage should still publish comment: %v", err)
	}
	failedComment := failedEvents.firstType("comment")
	if failedComment.Payload["memoryStatus"] != "failed" || failedComment.Payload["memoryPointer"] != nil {
		t.Fatalf("missing failed memory payload: %#v", failedComment.Payload)
	}
}

func TestAutopilotIntervals(t *testing.T) {
	cfg := AutopilotConfig{}
	if cfg.WorkerInterval() != 10*time.Second {
		t.Fatalf("unexpected default worker interval: %s", cfg.WorkerInterval())
	}
	if cfg.AgentPostInterval(0) != 120*time.Second {
		t.Fatalf("unexpected default post interval: %s", cfg.AgentPostInterval(0))
	}
	if cfg.AgentPostInterval(7*time.Second) != 7*time.Second {
		t.Fatalf("agent interval should win")
	}
}

func testAutopilot(events *fakeAutopilotEvents, agents []domain.Agent) Autopilot {
	return Autopilot{
		Agents:  fakeAgents{agents: agents},
		Events:  events,
		Compute: fakeCompute{},
		Storage: fakeStorage{},
	}
}

type fakeAgents struct {
	agents []domain.Agent
}

func (f fakeAgents) ListAgents(context.Context, int) ([]domain.Agent, error) {
	return append([]domain.Agent(nil), f.agents...), nil
}

func (fakeAgents) GetAgentByID(context.Context, string) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (fakeAgents) GetAgentByTokenID(context.Context, string) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (fakeAgents) GetAgentByAddress(context.Context, string) (domain.Agent, error) {
	return domain.Agent{}, nil
}

func (fakeAgents) UpsertAgent(context.Context, domain.Agent) error {
	return nil
}

type fakeAutopilotEvents struct {
	posts     []domain.Post
	existing  map[string]bool
	persisted []domain.SocialEvent
}

func newFakeAutopilotEvents(posts []domain.Post) *fakeAutopilotEvents {
	return &fakeAutopilotEvents{posts: posts, existing: map[string]bool{}}
}

func (f *fakeAutopilotEvents) UpsertSocialEvent(_ context.Context, event domain.SocialEvent) error {
	f.persisted = append(f.persisted, event)
	if key, ok := event.Payload["automationKey"].(string); ok {
		f.existing[key] = true
	}
	return nil
}

func (f *fakeAutopilotEvents) ListTimeline(context.Context, int) ([]domain.Post, error) {
	return f.posts, nil
}

func (f *fakeAutopilotEvents) ListAgentPosts(context.Context, string, int) ([]domain.Post, error) {
	return nil, nil
}

func (f *fakeAutopilotEvents) ListAgentSocialEvents(context.Context, string, int) ([]domain.SocialEvent, error) {
	return nil, nil
}

func (f *fakeAutopilotEvents) GetPostByID(_ context.Context, postID string) (domain.Post, error) {
	for _, post := range f.posts {
		if post.ID == postID {
			return post, nil
		}
	}
	return domain.Post{}, errors.New("post not found")
}

func (f *fakeAutopilotEvents) ListMentions(context.Context, string, int) ([]domain.SocialEvent, error) {
	return nil, nil
}

func (f *fakeAutopilotEvents) ListRecentPosts(_ context.Context, limit int) ([]domain.Post, error) {
	if limit > len(f.posts) {
		limit = len(f.posts)
	}
	return append([]domain.Post(nil), f.posts[:limit]...), nil
}

func (f *fakeAutopilotEvents) CountAutopilotActions(_ context.Context, postID string, actionType string) (int, error) {
	count := 0
	for _, event := range f.persisted {
		if event.Type == actionType && event.Payload["postId"] == postID && event.Payload["source"] == "autopilot" {
			count++
		}
	}
	return count, nil
}

func (f *fakeAutopilotEvents) HasAutomationKey(_ context.Context, automationKey string) (bool, error) {
	return f.existing[automationKey], nil
}

func (f *fakeAutopilotEvents) firstType(eventType string) domain.SocialEvent {
	for _, event := range f.persisted {
		if event.Type == eventType {
			return event
		}
	}
	return domain.SocialEvent{}
}

type fakeCompute struct {
	err error
}

func (f fakeCompute) Health(context.Context) error {
	return nil
}

func (f fakeCompute) RunLLM(context.Context, LLMRequest) (LLMResponse, error) {
	if f.err != nil {
		return LLMResponse{}, f.err
	}
	return LLMResponse{
		OutputText: "autopilot comment",
		Proof: domain.ProofOfInference{
			ModelID:    "test-model",
			InputHash:  "0xin",
			OutputHash: "0xout",
			TEESig:     "0xsig",
		},
	}, nil
}

func (f fakeCompute) RunImageGen(context.Context, ImageRequest) (ImageResponse, error) {
	return ImageResponse{}, nil
}

type fakeStorage struct {
	err error
}

func (f fakeStorage) Health(context.Context) error {
	return nil
}

func (f fakeStorage) UploadJSON(context.Context, any) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "storage://memory", nil
}

func (fakeStorage) UploadBytes(context.Context, string, []byte) (string, error) {
	return "", nil
}

func (fakeStorage) Fetch(context.Context, string) ([]byte, error) {
	return nil, nil
}
