package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AutopilotConfig struct {
	WorkerIntervalSeconds        int
	PostIntervalSeconds          int
	MaxPostsPerTick              int
	MaxLikesPerPost              int
	MaxCommentsPerPost           int
	BootstrapInitialDelaySeconds int
	BootstrapPostIntervalSeconds int
	BootstrapMaxImagePosts       int
	BootstrapMaxActiveAgents     int
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
	if c.BootstrapInitialDelaySeconds <= 0 {
		c.BootstrapInitialDelaySeconds = 10
	}
	if c.BootstrapPostIntervalSeconds <= 0 {
		c.BootstrapPostIntervalSeconds = 60
	}
	if c.BootstrapMaxImagePosts <= 0 {
		c.BootstrapMaxImagePosts = 3
	}
	if c.BootstrapMaxActiveAgents <= 0 {
		c.BootstrapMaxActiveAgents = 1
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

func (c AutopilotConfig) BootstrapInitialDelay() time.Duration {
	c = c.withDefaults()
	return time.Duration(c.BootstrapInitialDelaySeconds) * time.Second
}

func (c AutopilotConfig) BootstrapPostInterval() time.Duration {
	c = c.withDefaults()
	return time.Duration(c.BootstrapPostIntervalSeconds) * time.Second
}

type Autopilot struct {
	Agents   AgentRepository
	Events   AutopilotSocialEventRepository
	Metadata AgentMetadataRepository
	Compute  ZGComputeClient
	Storage  ZGStorageClient
	Config   AutopilotConfig
}

type AutopilotTickResult struct {
	PostsProcessed        int
	BootstrapPostsCreated int
	LikesCreated          int
	CommentsCreated       int
}

func (a Autopilot) Run(ctx context.Context) error {
	if !a.enabled() {
		log.Println("autopilot disabled: missing agents, events, compute, or storage dependency")
		<-ctx.Done()
		return ctx.Err()
	}

	interval := a.Config.withDefaults().WorkerInterval()
	log.Printf("autopilot worker starting: interval=%s maxPosts=%d maxLikes=%d maxComments=%d bootstrapDelay=%s bootstrapInterval=%s bootstrapMaxPosts=%d bootstrapMaxActiveAgents=%d",
		interval,
		a.Config.withDefaults().MaxPostsPerTick,
		a.Config.withDefaults().MaxLikesPerPost,
		a.Config.withDefaults().MaxCommentsPerPost,
		a.Config.withDefaults().BootstrapInitialDelay(),
		a.Config.withDefaults().BootstrapPostInterval(),
		a.Config.withDefaults().BootstrapMaxImagePosts,
		a.Config.withDefaults().BootstrapMaxActiveAgents,
	)

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
	agents, err := a.Agents.ListAgents(ctx, 100, 0)
	if err != nil {
		return AutopilotTickResult{}, err
	}
	sort.SliceStable(agents, func(i, j int) bool {
		return strings.ToLower(agents[i].ID) < strings.ToLower(agents[j].ID)
	})

	result := AutopilotTickResult{PostsProcessed: len(posts)}
	if err := a.createBootstrapPosts(ctx, agents, cfg, &result); err != nil {
		return result, err
	}
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

func (a Autopilot) createBootstrapPosts(ctx context.Context, agents []domain.Agent, cfg AutopilotConfig, result *AutopilotTickResult) error {
	now := time.Now().UTC()
	type candidate struct {
		agent      domain.Agent
		events     []domain.SocialEvent
		count      int
		latestPost time.Time
		dueAt      time.Time
	}

	active := make([]candidate, 0)
	newcomers := make([]candidate, 0)

	for _, agent := range agents {
		if agent.UpdatedAt.IsZero() {
			continue
		}
		events, err := a.Events.ListAgentSocialEvents(ctx, agent.ID, 100)
		if err != nil {
			return err
		}

		count, latestPostAt := bootstrapPostState(events)
		if count >= cfg.BootstrapMaxImagePosts {
			continue
		}

		dueAt := agent.UpdatedAt.Add(cfg.BootstrapInitialDelay())
		if !latestPostAt.IsZero() {
			dueAt = latestPostAt.Add(cfg.BootstrapPostInterval())
		}
		if now.Before(dueAt) {
			continue
		}

		item := candidate{
			agent:      agent,
			events:     events,
			count:      count,
			latestPost: latestPostAt,
			dueAt:      dueAt,
		}
		if count > 0 {
			active = append(active, item)
		} else {
			newcomers = append(newcomers, item)
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		if !active[i].latestPost.Equal(active[j].latestPost) {
			return active[i].latestPost.After(active[j].latestPost)
		}
		if !active[i].agent.UpdatedAt.Equal(active[j].agent.UpdatedAt) {
			return active[i].agent.UpdatedAt.After(active[j].agent.UpdatedAt)
		}
		return strings.ToLower(active[i].agent.ID) < strings.ToLower(active[j].agent.ID)
	})
	sort.SliceStable(newcomers, func(i, j int) bool {
		if !newcomers[i].agent.UpdatedAt.Equal(newcomers[j].agent.UpdatedAt) {
			return newcomers[i].agent.UpdatedAt.After(newcomers[j].agent.UpdatedAt)
		}
		return strings.ToLower(newcomers[i].agent.ID) < strings.ToLower(newcomers[j].agent.ID)
	})

	queue := active
	if len(queue) == 0 {
		queue = newcomers
	}
	if len(queue) == 0 {
		return nil
	}

	limit := cfg.BootstrapMaxActiveAgents
	if limit > len(queue) {
		limit = len(queue)
	}

	for _, item := range queue[:limit] {
		nextSequence := item.count + 1
		automationKey := AutomationKey(item.agent.ID, "bootstrap_post", strconv.Itoa(nextSequence))
		exists, err := a.Events.HasAutomationKey(ctx, automationKey)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := a.createBootstrapImagePost(ctx, item.agent, item.events, nextSequence, automationKey); err != nil {
			return err
		}
		result.BootstrapPostsCreated++
	}
	return nil
}

func bootstrapPostState(events []domain.SocialEvent) (count int, latest time.Time) {
	for _, event := range events {
		if event.Type != "post" {
			continue
		}
		if source, _ := event.Payload["source"].(string); source != "autopilot_bootstrap" {
			continue
		}
		count++
		if event.Timestamp.After(latest) {
			latest = event.Timestamp
		}
	}
	return count, latest
}

func (a Autopilot) createBootstrapImagePost(ctx context.Context, agent domain.Agent, events []domain.SocialEvent, sequence int, automationKey string) error {
	personality := strings.TrimSpace(agent.PersonalitySummary)
	if a.Metadata != nil && strings.TrimSpace(agent.MetadataPointer) != "" {
		metadata, err := a.Metadata.GetMetadata(ctx, agent.MetadataPointer)
		if err == nil && strings.TrimSpace(metadata.Prompt) != "" {
			personality = strings.TrimSpace(metadata.Prompt)
		}
	}

	memory := summarizeAutopilotEventsForPrompt(events)
	trigger := bootstrapImageTrigger(sequence)
	llm, err := a.Compute.RunLLM(ctx, LLMRequest{
		AgentID:     agent.ID,
		Personality: personality,
		Memory:      memory,
		Trigger:     trigger,
	})
	if err != nil {
		return err
	}

	createdAt := time.Now().UTC()
	memoryPointer, err := a.Storage.UploadJSON(ctx, map[string]any{
		"agentId":            agent.ID,
		"trigger":            trigger,
		"text":               llm.OutputText,
		"proof":              llm.Proof,
		"automationKey":      automationKey,
		"automationSequence": sequence,
		"createdAt":          createdAt.Format(time.RFC3339Nano),
		"source":             "autopilot_bootstrap",
	})
	if err != nil {
		return err
	}

	imagePrompt := buildAutopilotImagePrompt(personality, llm.OutputText)
	image, err := a.Compute.RunImageGen(ctx, ImageRequest{
		AgentID:     agent.ID,
		Personality: personality,
		Prompt:      imagePrompt,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{
		"text":               strings.TrimSpace(llm.OutputText),
		"proof":              llm.Proof,
		"memoryPointer":      memoryPointer,
		"source":             "autopilot_bootstrap",
		"automationKey":      automationKey,
		"automationSequence": sequence,
		"trigger":            trigger,
	}

	if image.ImageBase64 != "" {
		imageBytes, err := base64.StdEncoding.DecodeString(image.ImageBase64)
		if err != nil {
			return err
		}
		contentType := image.ContentType
		if contentType == "" {
			contentType = "image/jpeg"
		}
		imageRef, err := a.Storage.UploadBytes(ctx, contentType, imageBytes)
		if err != nil {
			return err
		}
		payload["imageRef"] = imageRef
		payload["imageProof"] = image.Proof
		payload["imageTeeVerified"] = image.TEEVerified
	}

	event := domain.SocialEvent{
		Type:      "post",
		AgentID:   agent.ID,
		Payload:   payload,
		Sig:       "compute",
		Timestamp: createdAt,
	}
	return a.persistAutopilotEvent(ctx, event)
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
		if err := a.persistAutopilotEvent(ctx, event); err != nil {
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
		if err := a.persistAutopilotEvent(ctx, event); err != nil {
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

func (a Autopilot) persistAutopilotEvent(ctx context.Context, event domain.SocialEvent) error {
	if event.BlobID == "" {
		event.BlobID = autopilotEventID(event)
	}
	event.ID = event.BlobID
	return a.Events.UpsertSocialEvent(ctx, event)
}

func (a Autopilot) enabled() bool {
	return a.Agents != nil && a.Events != nil && a.Compute != nil && a.Storage != nil
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

func autopilotEventID(event domain.SocialEvent) string {
	if key, ok := event.Payload["automationKey"].(string); ok && strings.TrimSpace(key) != "" {
		sum := sha256.Sum256([]byte(key))
		return "hybrid-autopilot-" + strings.ToLower(event.Type) + "-" + hex.EncodeToString(sum[:8])
	}
	sum := sha256.Sum256([]byte(event.AgentID + ":" + event.Type + ":" + event.Timestamp.Format(time.RFC3339Nano)))
	return "hybrid-autopilot-" + strings.ToLower(event.Type) + "-" + hex.EncodeToString(sum[:8])
}

func summarizeAutopilotEventsForPrompt(events []domain.SocialEvent) string {
	if len(events) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, event := range events {
		switch event.Type {
		case "post":
			if text, ok := event.Payload["text"].(string); ok && text != "" {
				builder.WriteString("Previous post: ")
				builder.WriteString(text)
				builder.WriteByte('\n')
			}
		case "comment", "like", "repost":
			builder.WriteString("Engagement ")
			builder.WriteString(event.Type)
			if postID, ok := event.Payload["postId"].(string); ok && postID != "" {
				builder.WriteString(" on ")
				builder.WriteString(postID)
			}
			if text, ok := event.Payload["text"].(string); ok && text != "" {
				builder.WriteString(": ")
				builder.WriteString(text)
			}
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func buildAutopilotImagePrompt(personality, postText string) string {
	persona := summarizeAutopilotText(personality, 160)
	if persona == "" {
		persona = "AI agent on a decentralized social feed"
	}
	post := strings.Join(strings.Fields(postText), " ")
	if len(post) > 200 {
		post = post[:200] + "..."
	}
	return "Editorial illustration for a social post by " + persona + ". Visualize: " + post
}

func summarizeAutopilotText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func bootstrapImageTrigger(sequence int) string {
	switch sequence {
	case 1:
		return "Publish the first public post that introduces this agent's voice with a clear, natural observation."
	case 2:
		return "Publish a fresh follow-up post that deepens this agent's perspective without repeating the previous post."
	case 3:
		return "Publish another distinct post that feels native to this agent's feed and expands the narrative naturally."
	default:
		return "Publish a fresh post that feels natural to this agent's feed."
	}
}
