package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

var (
	minGenerateOpsWei      = big.NewInt(10_000_000_000_000_000) // 0.01 OG
	minGenerateImageOpsWei = big.NewInt(20_000_000_000_000_000) // 0.02 OG
)

func (s Server) registerAPIRoutes(mux *stdhttp.ServeMux) {
	mux.HandleFunc("GET /capabilities", s.handleCapabilities)
	mux.HandleFunc("GET /agents", s.handleAgents)
	mux.HandleFunc("GET /agents/", s.handleAgentDetail)
	mux.HandleFunc("POST /agents/", s.handleAgentDetail)
	mux.HandleFunc("GET /external-agents", s.handleExternalAgents)
	mux.HandleFunc("GET /external-agents/", s.handleExternalAgentDetail)
	mux.HandleFunc("POST /external-agents/", s.handleExternalAgentDetail)
	mux.HandleFunc("PATCH /external-agents/", s.handleExternalAgentDetail)
	mux.HandleFunc("POST /external-agents/register", s.handleExternalAgentRegister)
	mux.HandleFunc("POST /external-agents/auth/challenge", s.handleExternalAgentChallenge)
	mux.HandleFunc("POST /external-agents/auth/verify", s.handleExternalAgentVerify)
	mux.HandleFunc("POST /external-actions", s.handleExternalAction)
	mux.HandleFunc("GET /timeline", s.handleTimeline)
	mux.HandleFunc("GET /storage", s.handleStorageFetch)
	mux.HandleFunc("POST /metadata", s.handleMetadata)
	mux.HandleFunc("GET /skills.md", s.handleSkills)
	mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /ws/timeline", s.handleTimelineWS)
	mux.HandleFunc("GET /posts/", s.handlePostDetail)
}

func (s Server) handlePostDetail(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/posts/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "post not found"})
		return
	}
	postID := parts[0]
	limit := parseLimit(r, 50)

	if len(parts) == 1 {
		// GET /posts/{id}
		post, err := s.Events.GetPostByID(r.Context(), postID)
		if err != nil {
			writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "post not found"})
			return
		}
		writeJSON(w, stdhttp.StatusOK, post)
		return
	}

	if len(parts) == 2 && parts[1] == "comments" {
		// GET /posts/{id}/comments
		comments, err := s.Events.ListPostComments(r.Context(), postID, limit)
		if err != nil {
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "failed to load comments"})
			return
		}
		writeJSON(w, stdhttp.StatusOK, comments)
		return
	}

	if len(parts) == 2 && parts[1] == "likes" {
		// GET /posts/{id}/likes
		likes, err := s.Events.ListPostLikes(r.Context(), postID, limit)
		if err != nil {
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "failed to load likes"})
			return
		}
		writeJSON(w, stdhttp.StatusOK, likes)
		return
	}

	writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "not found"})
}

func (s Server) handleMetadata(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.Metadata == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "metadata storage unavailable"})
		return
	}
	var request struct {
		Prompt             string `json:"prompt"`
		PersonalitySummary string `json:"personalitySummary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid metadata payload"})
		return
	}
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.PersonalitySummary = strings.TrimSpace(request.PersonalitySummary)
	if request.Prompt == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}
	if request.PersonalitySummary == "" {
		request.PersonalitySummary = summarizePrompt(request.Prompt)
	}
	updatedAt := time.Now().UTC()
	metadata := domain.AgentMetadata{
		Prompt:             request.Prompt,
		PersonalitySummary: request.PersonalitySummary,
		UpdatedAt:          updatedAt,
	}
	pointer, err := s.createMetadataPointer(r, metadata)
	if err != nil {
		log.Printf("create metadata pointer: %v", err)
		writeJSON(w, metadataPointerStatus(err), map[string]string{"error": err.Error()})
		return
	}
	metadata.MetadataPointer = pointer
	if err := s.Metadata.UpsertMetadata(r.Context(), metadata); err != nil {
		log.Printf("store metadata: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "metadata storage failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, metadata)
}

func (s Server) createMetadataPointer(r *stdhttp.Request, metadata domain.AgentMetadata) (string, error) {
	if s.Config.StubMode {
		return newStubMetadataPointer()
	}
	if s.Storage == nil {
		return "", errors.New("storage unavailable for real metadata mode")
	}
	pointer, err := s.Storage.UploadJSON(r.Context(), map[string]any{
		"prompt":             metadata.Prompt,
		"personalitySummary": metadata.PersonalitySummary,
		"updatedAt":          metadata.UpdatedAt.Format(time.RFC3339Nano),
		"source":             "mint-persona",
	})
	if err != nil {
		return "", fmt.Errorf("metadata storage upload failed: %w", err)
	}
	if strings.TrimSpace(pointer) == "" {
		return "", errors.New("metadata storage upload returned empty pointer")
	}
	return pointer, nil
}

func metadataPointerStatus(err error) int {
	if strings.Contains(err.Error(), "storage unavailable") {
		return stdhttp.StatusServiceUnavailable
	}
	return stdhttp.StatusBadGateway
}

func (s Server) handleAgents(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.Agents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "agent storage unavailable"})
		return
	}
	agents, err := s.Agents.ListAgents(r.Context(), 100)
	if err != nil {
		log.Printf("list agents from postgres: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "list agents failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, agents)
}

func (s Server) handleAgentDetail(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if r.Method == stdhttp.MethodPost && len(parts) == 2 && parts[1] == "generate-post" {
		s.handleGeneratePost(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "posts" {
		s.handleAgentPosts(w, r, parts[0])
		return
	}
	if r.Method == stdhttp.MethodPost && len(parts) == 2 && parts[1] == "follow" {
		s.handleAgentFollow(w, r, parts[0])
		return
	}
	if r.Method == stdhttp.MethodPost && len(parts) == 4 && parts[1] == "posts" && parts[3] == "actions" {
		s.handlePostAction(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "stats" {
		if s.Events == nil {
			writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
			return
		}
		followers, following, err := s.Events.GetAgentFollowStats(r.Context(), parts[0])
		if err != nil {
			log.Printf("get agent follow stats: %v", err)
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "get follow stats failed"})
			return
		}
		writeJSON(w, stdhttp.StatusOK, map[string]int{"followers": followers, "following": following})
		return
	}
	if len(parts) == 2 && parts[1] == "followers" {
		if s.Events == nil {
			writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
			return
		}
		followers, err := s.Events.ListAgentFollowers(r.Context(), parts[0], parseLimit(r, 20))
		if err != nil {
			log.Printf("list agent followers: %v", err)
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "list followers failed"})
			return
		}
		writeJSON(w, stdhttp.StatusOK, followers)
		return
	}
	if r.Method != stdhttp.MethodGet {
		writeJSON(w, stdhttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if len(parts) != 1 || parts[0] == "" {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if s.Agents != nil {
		agent, err := s.lookupAgent(r, parts[0])
		if err == nil {
			writeJSON(w, stdhttp.StatusOK, agent)
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("get agent from postgres: %v", err)
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "get agent failed"})
			return
		}
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}
	writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "agent not found"})
}

func (s Server) handleGeneratePost(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string) {
	agent, ok := s.resolveAgentOrError(w, r, agentID)
	if !ok {
		return
	}
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	if s.Compute == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "compute unavailable"})
		return
	}
	if s.Storage == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "storage unavailable"})
		return
	}

	var request struct {
		Trigger     string `json:"trigger"`
		WithImage   bool   `json:"withImage"`
		ImagePrompt string `json:"imagePrompt"`
		ActorAddress string `json:"actorAddress"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	request.ActorAddress = strings.TrimSpace(request.ActorAddress)
	if !isEVMAddress(request.ActorAddress) {
		writeJSON(w, stdhttp.StatusForbidden, map[string]string{"error": "owner wallet address is required to generate posts"})
		return
	}
	if !strings.EqualFold(request.ActorAddress, agent.OwnerAddress) {
		writeJSON(w, stdhttp.StatusForbidden, map[string]string{"error": "only the recorded owner wallet can generate posts for this agent"})
		return
	}
	trigger := strings.TrimSpace(request.Trigger)
	if trigger == "" {
		trigger = "manual profile run"
	}
	minRequired := minGenerateOpsWei
	if request.WithImage {
		minRequired = minGenerateImageOpsWei
	}
	if err := s.requireOperationalRunway(r.Context(), agent.AgentAddress, minRequired); err != nil {
		writeJSON(w, stdhttp.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	personality := agent.PersonalitySummary
	if s.Metadata != nil && agent.MetadataPointer != "" {
		metadata, err := s.Metadata.GetMetadata(r.Context(), agent.MetadataPointer)
		if err == nil && strings.TrimSpace(metadata.Prompt) != "" {
			personality = metadata.Prompt
		}
	}
	events, _ := s.Events.ListAgentSocialEvents(r.Context(), agent.ID, 20)
	memory := summarizeEventsForPrompt(events)
	llm, err := s.Compute.RunLLM(r.Context(), usecase.LLMRequest{
		AgentID:     agent.ID,
		Personality: personality,
		Memory:      memory,
		Trigger:     trigger,
	})
	if err != nil {
		log.Printf("generate post compute: %v", err)
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "compute failed"})
		return
	}

	createdAt := time.Now().UTC()
	memoryPointer, err := s.Storage.UploadJSON(r.Context(), map[string]any{
		"agentId":   agent.ID,
		"trigger":   trigger,
		"text":      llm.OutputText,
		"proof":     llm.Proof,
		"createdAt": createdAt,
		"source":    "generated",
	})
	if err != nil {
		log.Printf("generate post storage upload: %v", err)
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "storage upload failed"})
		return
	}

	var imageRef string
	var imageProof *domain.ProofOfInference
	var imageTEEVerified bool
	if request.WithImage {
		imagePrompt := strings.TrimSpace(request.ImagePrompt)
		if imagePrompt == "" {
			imagePrompt = buildImagePrompt(personality, llm.OutputText)
		}
		image, err := s.Compute.RunImageGen(r.Context(), usecase.ImageRequest{
			AgentID:     agent.ID,
			Personality: personality,
			Prompt:      imagePrompt,
		})
		if err != nil {
			log.Printf("generate post image: %v", err)
			writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image generation failed"})
			return
		}
		if image.ImageBase64 != "" {
			imageBytes, err := base64.StdEncoding.DecodeString(image.ImageBase64)
			if err != nil {
				log.Printf("generate post decode image: %v", err)
				writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image decode failed"})
				return
			}
			contentType := image.ContentType
			if contentType == "" {
				contentType = "image/jpeg"
			}
			pointer, err := s.Storage.UploadBytes(r.Context(), contentType, imageBytes)
			if err != nil {
				log.Printf("generate post upload image: %v", err)
				writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image upload failed"})
				return
			}
			imageRef = pointer
			proof := image.Proof
			imageProof = &proof
			imageTEEVerified = image.TEEVerified
		}
	}

	payload := map[string]any{
		"text":          llm.OutputText,
		"proof":         llm.Proof,
		"memoryPointer": memoryPointer,
		"source":        "generated",
	}
	if imageRef != "" {
		payload["imageRef"] = imageRef
		if imageProof != nil {
			payload["imageProof"] = imageProof
		}
		payload["imageTeeVerified"] = imageTEEVerified
	}
	event := domain.SocialEvent{
		BlobID:    fmt.Sprintf("hybrid-%s-%d", agent.ID, time.Now().UnixNano()),
		Type:      "post",
		AgentID:   agent.ID,
		Payload:   payload,
		Sig:       "compute",
		Timestamp: createdAt,
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		log.Printf("generate post persist: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "post persistence failed"})
		return
	}

	writeJSON(w, stdhttp.StatusCreated, domain.Post{
		ID:        event.BlobID,
		AgentID:   agent.ID,
		Text:      llm.OutputText,
		Proof:     llm.Proof,
		ImageRef:  imageRef,
		CreatedAt: createdAt,
	})
}

func (s Server) handlePostAction(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string, postID string) {
	agent, ok := s.resolveAgentOrError(w, r, agentID)
	if !ok {
		return
	}
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	writeJSON(w, stdhttp.StatusForbidden, map[string]string{
		"error": "human social actions are disabled; humans can mint, invest, and operate their own agent only",
	})
	return

	var request struct {
		Type         string `json:"type"`
		ActorAddress string `json:"actorAddress"`
		Text         string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid action payload"})
		return
	}
	request.Type = strings.ToLower(strings.TrimSpace(request.Type))
	if request.Type != "like" && request.Type != "comment" && request.Type != "repost" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "unsupported action type"})
		return
	}
	request.ActorAddress = strings.TrimSpace(request.ActorAddress)
	if request.ActorAddress == "" {
		request.ActorAddress = "anonymous"
	}
	request.Text = strings.TrimSpace(request.Text)
	if request.Type == "comment" && request.Text == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "comment text is required"})
		return
	}

	event := domain.SocialEvent{
		BlobID:  newEventID(request.Type, agent.ID),
		Type:    request.Type,
		AgentID: agent.ID,
		Payload: map[string]any{
			"postId":       postID,
			"actorAddress": request.ActorAddress,
			"text":         request.Text,
		},
		Sig:       "ui",
		Timestamp: time.Now().UTC(),
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		log.Printf("persist post action: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "action persistence failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, event)
}

func (s Server) handleTimeline(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	limit := parseLimit(r, 50)
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	posts, err := s.Events.ListTimeline(r.Context(), limit)
	if err != nil {
		log.Printf("list timeline from postgres: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "list timeline failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, posts)
}

func (s Server) handleAgentPosts(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string) {
	limit := parseLimit(r, 50)
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	resolvedAgentID := agentID
	if s.Agents != nil {
		agent, err := s.lookupAgent(r, agentID)
		if err == nil {
			resolvedAgentID = agent.ID
		} else if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("resolve agent for posts: %v", err)
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "resolve agent failed"})
			return
		}
	}
	posts, err := s.Events.ListAgentPosts(r.Context(), resolvedAgentID, limit)
	if err != nil {
		log.Printf("list agent posts from postgres: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "list agent posts failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, posts)
}

func (s Server) handleAgentFollow(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string) {
	agent, ok := s.resolveAgentOrError(w, r, agentID)
	if !ok {
		return
	}
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}

	var request struct {
		ActorAddress string `json:"actorAddress"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	request.ActorAddress = strings.TrimSpace(request.ActorAddress)
	if request.ActorAddress == "" {
		request.ActorAddress = "anonymous"
	}

	event := domain.SocialEvent{
		BlobID:  newEventID("follow", agent.ID),
		Type:    "follow",
		AgentID: "human", // Human is acting
		Payload: map[string]any{
			"targetAgentId": agent.ID,
			"actorAddress":  request.ActorAddress,
		},
		Sig:       "ui",
		Timestamp: time.Now().UTC(),
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		log.Printf("persist follow action: %v", err)
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "follow persistence failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, event)
}

func (s Server) lookupAgent(r *stdhttp.Request, id string) (domain.Agent, error) {
	if isEVMAddress(id) {
		return s.Agents.GetAgentByAddress(r.Context(), id)
	}
	return s.Agents.GetAgentByID(r.Context(), id)
}

func (s Server) resolveAgentOrError(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string) (domain.Agent, bool) {
	if s.Agents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "agent storage unavailable"})
		return domain.Agent{}, false
	}
	agent, err := s.lookupAgent(r, agentID)
	if err == nil {
		return agent, true
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "agent not found"})
		return domain.Agent{}, false
	}
	log.Printf("resolve agent: %v", err)
	writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "resolve agent failed"})
	return domain.Agent{}, false
}

func (s Server) persistPost(r *stdhttp.Request, agentID string, text string, proof domain.ProofOfInference, source string) (domain.Post, error) {
	now := time.Now().UTC()
	event := domain.SocialEvent{
		BlobID:  newEventID("post", agentID),
		Type:    "post",
		AgentID: agentID,
		Payload: map[string]any{
			"text":   text,
			"proof":  proof,
			"source": source,
		},
		Sig:       source,
		Timestamp: now,
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		return domain.Post{}, err
	}
	return domain.Post{
		ID:        event.BlobID,
		AgentID:   agentID,
		Text:      text,
		Proof:     proof,
		CreatedAt: now,
	}, nil
}

func newEventID(prefix string, agentID string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, strings.ToLower(strings.TrimPrefix(agentID, "0x")), time.Now().UnixNano())
}

func summarizeEventsForPrompt(events []domain.SocialEvent) string {
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

func isEVMAddress(value string) bool {
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return false
	}
	for _, char := range value[2:] {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') {
			continue
		}
		return false
	}
	return true
}

func parseLimit(r *stdhttp.Request, fallback int) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func writeJSON(w stdhttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newStubMetadataPointer() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "stub://metadata/" + hex.EncodeToString(bytes[:]), nil
}

func (s Server) handleStorageFetch(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	pointer := strings.TrimSpace(r.URL.Query().Get("pointer"))
	if pointer == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "pointer query param required"})
		return
	}
	if s.Storage == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "storage unavailable"})
		return
	}
	bytes, err := s.Storage.Fetch(r.Context(), pointer)
	if err != nil {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "pointer not found"})
		return
	}
	contentType := stdhttp.DetectContentType(bytes)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write(bytes)
}

func buildImagePrompt(personality, postText string) string {
	persona := summarizePrompt(personality)
	if persona == "" {
		persona = "AI agent on a decentralized social feed"
	}
	post := strings.Join(strings.Fields(postText), " ")
	if len(post) > 200 {
		post = post[:200] + "..."
	}
	return "Editorial illustration for a social post by " + persona + ". Visualize: " + post
}

func summarizePrompt(prompt string) string {
	const max = 160
	prompt = strings.Join(strings.Fields(prompt), " ")
	if len(prompt) <= max {
		return prompt
	}
	return prompt[:max] + "..."
}

func (s Server) requireOperationalRunway(ctx context.Context, treasuryAddress string, minimum *big.Int) error {
	if s.Config.StubMode {
		return nil
	}
	if !isEVMAddress(treasuryAddress) {
		return errors.New("agent treasury address is unavailable")
	}
	if strings.TrimSpace(s.Config.OGRPCURL) == "" {
		return errors.New("chain rpc is unavailable for ops runway checks")
	}

	balance, err := fetchOperationalBalance(ctx, s.Config.OGRPCURL, treasuryAddress)
	if err != nil {
		return fmt.Errorf("failed to verify operational runway: %w", err)
	}
	if balance.Cmp(minimum) < 0 {
		return fmt.Errorf("insufficient operational balance: treasury needs at least %s wei before manual generation", minimum.String())
	}
	return nil
}

func fetchOperationalBalance(ctx context.Context, rpcURL string, treasuryAddress string) (*big.Int, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params": []any{
			map[string]any{
				"to":   treasuryAddress,
				"data": "0xf865216e",
			},
			"latest",
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, rpcURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := stdhttp.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, fmt.Errorf("rpc %d: %s", response.Error.Code, response.Error.Message)
	}
	if response.Result == "" {
		return nil, errors.New("empty eth_call result")
	}
	hexValue := strings.TrimPrefix(response.Result, "0x")
	if hexValue == "" {
		return big.NewInt(0), nil
	}
	value, ok := new(big.Int).SetString(hexValue, 16)
	if !ok {
		return nil, errors.New("invalid operational balance hex result")
	}
	return value, nil
}
