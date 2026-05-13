package usecase

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AutopilotConfig struct {
	WorkerIntervalSeconds int
	PostIntervalSeconds   int
	MaxPostsPerTick       int
	MaxLikesPerPost       int
	MaxCommentsPerPost    int
}

func (c AutopilotConfig) withDefaults() AutopilotConfig {
	if c.WorkerIntervalSeconds <= 0 {
		c.WorkerIntervalSeconds = 10
	}
	if c.PostIntervalSeconds <= 0 {
		c.PostIntervalSeconds = 120
	}
	if c.MaxPostsPerTick <= 0 {
		c.MaxPostsPerTick = 5
	}
	if c.MaxLikesPerPost <= 0 {
		c.MaxLikesPerPost = 3
	}
	if c.MaxCommentsPerPost <= 0 {
		c.MaxCommentsPerPost = 2
	}
	return c
}

func (c AutopilotConfig) WorkerInterval() time.Duration {
	c = c.withDefaults()
	return time.Duration(c.WorkerIntervalSeconds) * time.Second
}

func (c AutopilotConfig) DefaultPostInterval() time.Duration {
	c = c.withDefaults()
	return time.Duration(c.PostIntervalSeconds) * time.Second
}

func (c AutopilotConfig) AgentPostInterval(agentInterval time.Duration) time.Duration {
	if agentInterval > 0 {
		return agentInterval
	}
	return c.DefaultPostInterval()
}

type Autopilot struct {
	Agents  AgentRepository
	Events  AutopilotSocialEventRepository
	Compute ZGComputeClient
	DA      ZGDAClient
	Storage ZGStorageClient
	Config  AutopilotConfig
}

type AutopilotTickResult struct {
	PostsProcessed  int
	LikesCreated    int
	CommentsCreated int
}

func (a Autopilot) Run(ctx context.Context) error {
	if !a.enabled() {
		log.Println("autopilot disabled: missing agents, events, compute, DA, or storage dependency")
		<-ctx.Done()
		return ctx.Err()
	}

	interval := a.Config.withDefaults().WorkerInterval()
	log.Printf("autopilot worker starting: interval=%s maxPosts=%d maxLikes=%d maxComments=%d",
		interval, a.Config.withDefaults().MaxPostsPerTick, a.Config.withDefaults().MaxLikesPerPost, a.Config.withDefaults().MaxCommentsPerPost)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := a.Tick(ctx); err != nil {
			log.Printf("autopilot tick error: %v", err)
		}
		select {
		case <-ctx.Done():
			log.Println("autopilot worker stopped")
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a Autopilot) Tick(ctx context.Context) (AutopilotTickResult, error) {
	if !a.enabled() {
		return AutopilotTickResult{}, fmt.Errorf("autopilot disabled: missing dependency")
	}

	cfg := a.Config.withDefaults()
	posts, err := a.Events.ListRecentPosts(ctx, cfg.MaxPostsPerTick)
	if err != nil {
		return AutopilotTickResult{}, err
	}
	agents, err := a.Agents.ListAgents(ctx, 100)
	if err != nil {
		return AutopilotTickResult{}, err
	}
	sort.SliceStable(agents, func(i, j int) bool {
		return strings.ToLower(agents[i].ID) < strings.ToLower(agents[j].ID)
	})

	result := AutopilotTickResult{PostsProcessed: len(posts)}
	for _, post := range posts {
		candidates := eligibleAgents(agents, post.AgentID)
		if err := a.createLikes(ctx, post, candidates, cfg, &result); err != nil {
			return result, err
		}
		if err := a.createComments(ctx, post, candidates, cfg, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (a Autopilot) createLikes(ctx context.Context, post domain.Post, candidates []domain.Agent, cfg AutopilotConfig, result *AutopilotTickResult) error {
	count, err := a.Events.CountAutopilotActions(ctx, post.ID, "like")
	if err != nil {
		return err
	}
	for _, agent := range candidates {
		if count >= cfg.MaxLikesPerPost {
			return nil
		}
		key := AutomationKey(agent.ID, "like", post.ID)
		exists, err := a.Events.HasAutomationKey(ctx, key)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		event := domain.SocialEvent{
			Type:    "like",
			AgentID: agent.ID,
			Payload: map[string]any{
				"actorAgentId":  agent.ID,
				"postId":        post.ID,
				"targetPostId":  post.ID,
				"source":        "autopilot",
				"automationKey": key,
			},
			Sig:       "autopilot",
			Timestamp: time.Now().UTC(),
		}
		if err := a.publishAndPersist(ctx, event); err != nil {
			return err
		}
		count++
		result.LikesCreated++
	}
	return nil
}

func (a Autopilot) createComments(ctx context.Context, post domain.Post, candidates []domain.Agent, cfg AutopilotConfig, result *AutopilotTickResult) error {
	count, err := a.Events.CountAutopilotActions(ctx, post.ID, "comment")
	if err != nil {
		return err
	}
	for _, agent := range candidates {
		if count >= cfg.MaxCommentsPerPost {
			return nil
		}
		key := AutomationKey(agent.ID, "comment", post.ID)
		exists, err := a.Events.HasAutomationKey(ctx, key)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		llm, err := a.Compute.RunLLM(ctx, LLMRequest{
			AgentID:     agent.ID,
			Personality: agent.PersonalitySummary,
			Memory:      fmt.Sprintf("Recent target post by %s: %s", post.AgentID, post.Text),
			Trigger:     "autopilot_comment",
		})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"actorAgentId":  agent.ID,
			"postId":        post.ID,
			"targetPostId":  post.ID,
			"text":          strings.TrimSpace(llm.OutputText),
			"source":        "autopilot",
			"automationKey": key,
			"proof":         llm.Proof,
		}
		a.appendCommentMemory(ctx, agent, post, llm.OutputText, payload)

		event := domain.SocialEvent{
			Type:      "comment",
			AgentID:   agent.ID,
			Payload:   payload,
			Sig:       "autopilot",
			Timestamp: time.Now().UTC(),
		}
		if err := a.publishAndPersist(ctx, event); err != nil {
			return err
		}
		count++
		result.CommentsCreated++
	}
	return nil
}

func (a Autopilot) appendCommentMemory(ctx context.Context, agent domain.Agent, post domain.Post, comment string, payload map[string]any) {
	pointer, err := a.Storage.UploadJSON(ctx, map[string]any{
		"type":         "autopilot_comment",
		"agentId":      agent.ID,
		"targetPostId": post.ID,
		"targetText":   post.Text,
		"comment":      comment,
		"createdAt":    time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		payload["memoryStatus"] = "failed"
		payload["memoryError"] = err.Error()
		return
	}
	payload["memoryStatus"] = "updated"
	payload["memoryPointer"] = pointer
}

func (a Autopilot) publishAndPersist(ctx context.Context, event domain.SocialEvent) error {
	blobID, err := a.DA.PublishSocialEvent(ctx, event)
	if err != nil {
		return err
	}
	event.BlobID = blobID
	event.ID = blobID
	return a.Events.UpsertSocialEvent(ctx, event)
}

func (a Autopilot) enabled() bool {
	return a.Agents != nil && a.Events != nil && a.Compute != nil && a.DA != nil && a.Storage != nil
}

func eligibleAgents(agents []domain.Agent, authorID string) []domain.Agent {
	out := make([]domain.Agent, 0, len(agents))
	for _, agent := range agents {
		if strings.EqualFold(agent.ID, authorID) {
			continue
		}
		out = append(out, agent)
	}
	return out
}

func AutomationKey(agentID string, actionType string, targetPostID string) string {
	return strings.ToLower(strings.TrimSpace(agentID)) + ":" + strings.ToLower(strings.TrimSpace(actionType)) + ":" + strings.TrimSpace(targetPostID)
}
