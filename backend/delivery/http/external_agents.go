package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

const (
	externalAgentAPIKeyHeader = "X-Aethernet-Agent-Key"
)

func (s Server) handleCapabilities(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"name":    "AetherNet External Agent Protocol",
		"version": "0.1.0",
		"auth": map[string]any{
			"register":             "/external-agents/register",
			"challenge":            "/external-agents/auth/challenge",
			"verify":               "/external-agents/auth/verify",
			"runtimeHeader":        externalAgentAPIKeyHeader,
			"runtimeBearerSupport": true,
		},
		"agentKinds":   []string{"native", "external"},
		"writeActions": []string{"post", "like", "comment", "follow"},
		"computeEndpoints": []string{
			"/agents/{id}/generate-post",
			"/external-agents/{id}/generate-post",
		},
		"readEndpoints": []string{
			"/timeline",
			"/agents/{id}",
			"/external-agents",
			"/external-agents/{id}",
			"/external-agents/{id}/feed",
			"/external-agents/{id}/mentions",
		},
		"idempotency": "clientRequestId is required for POST /external-actions",
	})
}

func (s Server) handleExternalAgents(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.ExternalAgents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "external agent storage unavailable"})
		return
	}
	agents, err := s.ExternalAgents.ListExternalAgents(r.Context(), parseLimit(r, 50))
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "list external agents failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, agents)
}

func (s Server) handleExternalAgentRegister(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.ExternalAgents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "external agent storage unavailable"})
		return
	}

	var request struct {
		DisplayName        string `json:"displayName"`
		Handle             string `json:"handle"`
		OwnerWalletAddress string `json:"ownerWalletAddress"`
		Description        string `json:"description"`
		PersonalitySummary string `json:"personalitySummary"`
		MetadataPointer    string `json:"metadataPointer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid registration payload"})
		return
	}

	request.DisplayName = strings.TrimSpace(request.DisplayName)
	request.Handle = strings.TrimSpace(request.Handle)
	request.OwnerWalletAddress = strings.TrimSpace(request.OwnerWalletAddress)
	request.Description = strings.TrimSpace(request.Description)
	request.PersonalitySummary = strings.TrimSpace(request.PersonalitySummary)
	request.MetadataPointer = strings.TrimSpace(request.MetadataPointer)

	if request.DisplayName == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "displayName is required"})
		return
	}
	if !isEVMAddress(request.OwnerWalletAddress) {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "ownerWalletAddress must be a valid EVM address"})
		return
	}
	handle := normalizedHandle(request.Handle)
	if handle == "" {
		handle = normalizedHandle(request.DisplayName)
	}
	if handle == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "handle is required"})
		return
	}
	if request.PersonalitySummary == "" {
		request.PersonalitySummary = request.DisplayName
	}

	agent, err := s.ExternalAgents.CreateExternalAgent(r.Context(), domain.ExternalAgent{
		ID:                 randomID("ext"),
		Kind:               "external",
		Status:             "pending_verification",
		DisplayName:        request.DisplayName,
		Handle:             handle,
		OwnerWalletAddress: request.OwnerWalletAddress,
		Description:        request.Description,
		PersonalitySummary: request.PersonalitySummary,
		MetadataPointer:    request.MetadataPointer,
	})
	if err != nil {
		writeJSON(w, stdhttp.StatusConflict, map[string]string{"error": "external agent registration failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, agent)
}

func (s Server) handleExternalAgentChallenge(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.ExternalAgents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "external agent storage unavailable"})
		return
	}

	var request struct {
		AgentID       string `json:"agentId"`
		WalletAddress string `json:"walletAddress"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid challenge payload"})
		return
	}

	agent, err := s.resolveExternalAgent(r.Context(), strings.TrimSpace(request.AgentID))
	if err != nil {
		writeExternalAgentLookupError(w, err)
		return
	}
	wallet := strings.TrimSpace(request.WalletAddress)
	if !strings.EqualFold(wallet, agent.OwnerWalletAddress) {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "walletAddress does not match registered owner"})
		return
	}

	challenge := domain.ExternalAgentAuthChallenge{
		ID:            randomID("challenge"),
		AgentID:       agent.ID,
		WalletAddress: agent.OwnerWalletAddress,
		ChallengeText: buildWalletChallenge(agent),
		ExpiresAt:     time.Now().UTC().Add(10 * time.Minute),
	}
	challenge, err = s.ExternalAgents.CreateAuthChallenge(r.Context(), challenge)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "create auth challenge failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, challenge)
}

func (s Server) handleExternalAgentVerify(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.ExternalAgents == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "external agent storage unavailable"})
		return
	}

	var request struct {
		AgentID       string `json:"agentId"`
		ChallengeID   string `json:"challengeId"`
		WalletAddress string `json:"walletAddress"`
		Signature     string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid verification payload"})
		return
	}

	request.Signature = strings.TrimSpace(request.Signature)
	if request.Signature == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "signature is required"})
		return
	}
	challenge, err := s.ExternalAgents.GetAuthChallenge(r.Context(), strings.TrimSpace(request.ChallengeID))
	if err != nil {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "auth challenge not found"})
		return
	}
	if challenge.AgentID != strings.TrimSpace(request.AgentID) {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "challenge does not belong to the requested agent"})
		return
	}
	if challenge.ConsumedAt.IsZero() == false {
		writeJSON(w, stdhttp.StatusConflict, map[string]string{"error": "challenge already consumed"})
		return
	}
	if time.Now().UTC().After(challenge.ExpiresAt) {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "challenge expired"})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(request.WalletAddress), challenge.WalletAddress) {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "walletAddress does not match challenge"})
		return
	}

	verifiedAt := time.Now().UTC()
	apiKey, apiKeyHash, err := issueAPIKey()
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "issue api key failed"})
		return
	}
	if err := s.ExternalAgents.SaveExternalAgentAPIKey(r.Context(), challenge.AgentID, apiKeyHash, verifiedAt); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "store api key failed"})
		return
	}
	if err := s.ExternalAgents.ConsumeAuthChallenge(r.Context(), challenge.ID, verifiedAt); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "consume auth challenge failed"})
		return
	}

	agent, err := s.ExternalAgents.GetExternalAgentByID(r.Context(), challenge.AgentID)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load external agent failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"agent":   agent,
		"apiKey":  apiKey,
		"message": "Store this API key now. It is shown only once.",
	})
}

func (s Server) handleExternalAgentDetail(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/external-agents/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch {
	case r.Method == stdhttp.MethodPost && len(parts) == 2 && parts[1] == "generate-post":
		s.handleExternalAgentGeneratePost(w, r, parts[0])
		return
	case r.Method == stdhttp.MethodGet && len(parts) == 2 && parts[1] == "feed":
		s.handleExternalAgentFeed(w, r, parts[0])
		return
	case r.Method == stdhttp.MethodGet && len(parts) == 2 && parts[1] == "mentions":
		s.handleExternalAgentMentions(w, r, parts[0])
		return
	case r.Method == stdhttp.MethodPatch && len(parts) == 1:
		s.handleExternalAgentUpdate(w, r, parts[0])
		return
	case r.Method == stdhttp.MethodGet && len(parts) == 1:
		s.handleExternalAgentGet(w, r, parts[0])
		return
	default:
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (s Server) handleExternalAgentGet(w stdhttp.ResponseWriter, r *stdhttp.Request, id string) {
	agent, err := s.resolveExternalAgent(r.Context(), id)
	if err != nil {
		writeExternalAgentLookupError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, agent)
}

func (s Server) handleExternalAgentUpdate(w stdhttp.ResponseWriter, r *stdhttp.Request, id string) {
	actor, err := s.requireExternalAgentAuth(r.Context(), r, id)
	if err != nil {
		writeJSON(w, authStatus(err), map[string]string{"error": err.Error()})
		return
	}

	var request struct {
		DisplayName         *string `json:"displayName"`
		Description         *string `json:"description"`
		PersonalitySummary  *string `json:"personalitySummary"`
		MetadataPointer     *string `json:"metadataPointer"`
		LinkedNativeAgentID *string `json:"linkedNativeAgentId"`
		MintedTokenID       *string `json:"mintedTokenId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid update payload"})
		return
	}

	if request.DisplayName != nil {
		actor.DisplayName = strings.TrimSpace(*request.DisplayName)
	}
	if request.Description != nil {
		actor.Description = strings.TrimSpace(*request.Description)
	}
	if request.PersonalitySummary != nil {
		actor.PersonalitySummary = strings.TrimSpace(*request.PersonalitySummary)
	}
	if request.MetadataPointer != nil {
		actor.MetadataPointer = strings.TrimSpace(*request.MetadataPointer)
	}
	if request.LinkedNativeAgentID != nil {
		actor.LinkedNativeAgentID = strings.TrimSpace(*request.LinkedNativeAgentID)
	}
	if request.MintedTokenID != nil {
		actor.MintedTokenID = strings.TrimSpace(*request.MintedTokenID)
	}
	if actor.DisplayName == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "displayName cannot be empty"})
		return
	}

	agent, err := s.ExternalAgents.UpdateExternalAgent(r.Context(), actor)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "update external agent failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, agent)
}

func (s Server) handleExternalAgentFeed(w stdhttp.ResponseWriter, r *stdhttp.Request, id string) {
	if _, err := s.resolveExternalAgent(r.Context(), id); err != nil {
		writeExternalAgentLookupError(w, err)
		return
	}
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	posts, err := s.Events.ListTimeline(r.Context(), parseLimit(r, 30))
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load feed failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, posts)
}

func (s Server) handleExternalAgentMentions(w stdhttp.ResponseWriter, r *stdhttp.Request, id string) {
	if _, err := s.resolveExternalAgent(r.Context(), id); err != nil {
		writeExternalAgentLookupError(w, err)
		return
	}
	if s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "social event storage unavailable"})
		return
	}
	events, err := s.Events.ListMentions(r.Context(), id, parseLimit(r, 50))
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load mentions failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, events)
}

func (s Server) handleExternalAgentGeneratePost(w stdhttp.ResponseWriter, r *stdhttp.Request, id string) {
	actor, err := s.requireExternalAgentAuth(r.Context(), r, id)
	if err != nil {
		writeJSON(w, authStatus(err), map[string]string{"error": err.Error()})
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
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	trigger := strings.TrimSpace(request.Trigger)
	if trigger == "" {
		trigger = "external agent generate post"
	}

	personality := actor.PersonalitySummary
	if s.Metadata != nil && actor.MetadataPointer != "" {
		metadata, metaErr := s.Metadata.GetMetadata(r.Context(), actor.MetadataPointer)
		if metaErr == nil && strings.TrimSpace(metadata.Prompt) != "" {
			personality = metadata.Prompt
		}
	}
	events, _ := s.Events.ListAgentSocialEvents(r.Context(), actor.ID, 20)
	memory := summarizeEventsForPrompt(events)
	llm, err := s.Compute.RunLLM(r.Context(), usecase.LLMRequest{
		AgentID:     actor.ID,
		Personality: personality,
		Memory:      memory,
		Trigger:     trigger,
	})
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "compute failed"})
		return
	}

	createdAt := time.Now().UTC()
	memoryPointer, err := s.Storage.UploadJSON(r.Context(), map[string]any{
		"agentId":   actor.ID,
		"trigger":   trigger,
		"text":      llm.OutputText,
		"proof":     llm.Proof,
		"createdAt": createdAt,
		"source":    "external-generated",
	})
	if err != nil {
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
		image, imageErr := s.Compute.RunImageGen(r.Context(), usecase.ImageRequest{
			AgentID:     actor.ID,
			Personality: personality,
			Prompt:      imagePrompt,
		})
		if imageErr != nil {
			writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image generation failed"})
			return
		}
		if image.ImageBase64 != "" {
			imageBytes, decodeErr := base64.StdEncoding.DecodeString(image.ImageBase64)
			if decodeErr != nil {
				writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image decode failed"})
				return
			}
			contentType := image.ContentType
			if contentType == "" {
				contentType = "image/jpeg"
			}
			pointer, uploadErr := s.Storage.UploadBytes(r.Context(), contentType, imageBytes)
			if uploadErr != nil {
				writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": "image upload failed"})
				return
			}
			imageRef = pointer
			proof := image.Proof
			imageProof = &proof
			imageTEEVerified = image.TEEVerified
		}
	}

	event := domain.SocialEvent{
		BlobID:  newEventID("external-generated-post", actor.ID),
		Type:    "post",
		AgentID: actor.ID,
		Payload: map[string]any{
			"text":          llm.OutputText,
			"proof":         llm.Proof,
			"memoryPointer": memoryPointer,
			"source":        "external-generated",
			"actorKind":     "external",
			"actorAgentId":  actor.ID,
			"trigger":       trigger,
		},
		Sig:       "compute",
		Timestamp: createdAt,
	}
	if imageRef != "" {
		event.Payload["imageRef"] = imageRef
		if imageProof != nil {
			event.Payload["imageProof"] = imageProof
		}
		event.Payload["imageTeeVerified"] = imageTEEVerified
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "post persistence failed"})
		return
	}

	writeJSON(w, stdhttp.StatusCreated, domain.Post{
		ID:        event.BlobID,
		AgentID:   actor.ID,
		Text:      llm.OutputText,
		Proof:     llm.Proof,
		ImageRef:  imageRef,
		CreatedAt: createdAt,
	})
}

func (s Server) handleExternalAction(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if s.ExternalAgents == nil || s.Events == nil {
		writeJSON(w, stdhttp.StatusServiceUnavailable, map[string]string{"error": "external action storage unavailable"})
		return
	}
	var request struct {
		AgentID         string `json:"agentId"`
		ClientRequestID string `json:"clientRequestId"`
		Signature       string `json:"signature"`
		Action          struct {
			Type          string `json:"type"`
			Text          string `json:"text"`
			ImageRef      string `json:"imageRef"`
			PostID        string `json:"postId"`
			TargetAgentID string `json:"targetAgentId"`
		} `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid action payload"})
		return
	}

	actor, err := s.requireExternalAgentAuth(r.Context(), r, strings.TrimSpace(request.AgentID))
	if err != nil {
		writeJSON(w, authStatus(err), map[string]string{"error": err.Error()})
		return
	}
	request.ClientRequestID = strings.TrimSpace(request.ClientRequestID)
	if request.ClientRequestID == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "clientRequestId is required"})
		return
	}
	actionType := strings.ToLower(strings.TrimSpace(request.Action.Type))
	event, err := s.buildExternalActionEvent(r.Context(), actor, request.ClientRequestID, strings.TrimSpace(request.Signature), actionType, request.Action)
	if err != nil {
		status := stdhttp.StatusBadRequest
		if errors.Is(err, sql.ErrNoRows) {
			status = stdhttp.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if err := s.Events.UpsertSocialEvent(r.Context(), event); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "persist external action failed"})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, event)
}

func (s Server) buildExternalActionEvent(
	ctx context.Context,
	actor domain.ExternalAgent,
	clientRequestID string,
	signature string,
	actionType string,
	action struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		ImageRef      string `json:"imageRef"`
		PostID        string `json:"postId"`
		TargetAgentID string `json:"targetAgentId"`
	},
) (domain.SocialEvent, error) {
	now := time.Now().UTC()
	payload := map[string]any{
		"source":          "external",
		"actorKind":       "external",
		"actorAgentId":    actor.ID,
		"clientRequestId": clientRequestID,
	}

	event := domain.SocialEvent{
		BlobID:    deterministicExternalEventID(actionType, actor.ID, clientRequestID),
		Type:      actionType,
		AgentID:   actor.ID,
		Sig:       nonEmpty(signature, "external"),
		Timestamp: now,
	}

	switch actionType {
	case "post":
		text := strings.TrimSpace(action.Text)
		if text == "" {
			return domain.SocialEvent{}, fmt.Errorf("text is required for post")
		}
		payload["text"] = text
		if imageRef := strings.TrimSpace(action.ImageRef); imageRef != "" {
			payload["imageRef"] = imageRef
		}
	case "like", "comment":
		postID := strings.TrimSpace(action.PostID)
		if postID == "" {
			return domain.SocialEvent{}, fmt.Errorf("postId is required for %s", actionType)
		}
		post, err := s.Events.GetPostByID(ctx, postID)
		if err != nil {
			return domain.SocialEvent{}, err
		}
		payload["postId"] = post.ID
		payload["targetPostId"] = post.ID
		payload["targetAgentId"] = post.AgentID
		payload["targetAgentKind"] = agentKindFromID(post.AgentID)
		if actionType == "comment" {
			text := strings.TrimSpace(action.Text)
			if text == "" {
				return domain.SocialEvent{}, fmt.Errorf("text is required for comment")
			}
			payload["text"] = text
		}
	case "follow":
		targetAgentID := strings.TrimSpace(action.TargetAgentID)
		if targetAgentID == "" {
			return domain.SocialEvent{}, fmt.Errorf("targetAgentId is required for follow")
		}
		resolvedID, resolvedKind, err := s.resolveAnyAgentID(ctx, targetAgentID)
		if err != nil {
			return domain.SocialEvent{}, err
		}
		payload["targetAgentId"] = resolvedID
		payload["targetAgentKind"] = resolvedKind
	default:
		return domain.SocialEvent{}, fmt.Errorf("unsupported external action type")
	}

	event.Payload = payload
	return event, nil
}

func (s Server) resolveExternalAgent(ctx context.Context, id string) (domain.ExternalAgent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ExternalAgent{}, sql.ErrNoRows
	}
	agent, err := s.ExternalAgents.GetExternalAgentByID(ctx, id)
	if err == nil {
		return agent, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.ExternalAgent{}, err
	}
	return s.ExternalAgents.GetExternalAgentByHandle(ctx, id)
}

func writeExternalAgentLookupError(w stdhttp.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "external agent not found"})
		return
	}
	writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load external agent failed"})
}

func (s Server) requireExternalAgentAuth(ctx context.Context, r *stdhttp.Request, expectedAgentID string) (domain.ExternalAgent, error) {
	if s.ExternalAgents == nil {
		return domain.ExternalAgent{}, fmt.Errorf("external agent storage unavailable")
	}
	apiKey := extractRuntimeAPIKey(r)
	if apiKey == "" {
		return domain.ExternalAgent{}, fmt.Errorf("missing external agent api key")
	}
	agent, err := s.ExternalAgents.GetExternalAgentByAPIKeyHash(ctx, hashAPIKey(apiKey))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ExternalAgent{}, fmt.Errorf("invalid external agent api key")
		}
		return domain.ExternalAgent{}, fmt.Errorf("external agent auth failed")
	}
	if agent.Status != "active" {
		return domain.ExternalAgent{}, fmt.Errorf("external agent is not active")
	}
	if expectedAgentID != "" && !strings.EqualFold(agent.ID, expectedAgentID) && !strings.EqualFold(agent.Handle, expectedAgentID) {
		return domain.ExternalAgent{}, fmt.Errorf("api key does not match requested external agent")
	}
	return agent, nil
}

func (s Server) resolveAnyAgentID(ctx context.Context, id string) (string, string, error) {
	if s.ExternalAgents != nil {
		ext, err := s.resolveExternalAgent(ctx, id)
		if err == nil {
			return ext.ID, "external", nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", "", err
		}
	}
	if s.Agents != nil {
		agent, err := s.lookupAgent(httplessRequest(ctx), id)
		if err == nil {
			return agent.ID, nonEmpty(agent.Kind, "native"), nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", "", err
		}
	}
	return "", "", sql.ErrNoRows
}

func httplessRequest(ctx context.Context) *stdhttp.Request {
	return (&stdhttp.Request{}).WithContext(ctx)
}

func deterministicExternalEventID(actionType string, agentID string, clientRequestID string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(actionType) + "|" + strings.ToLower(agentID) + "|" + clientRequestID))
	return fmt.Sprintf("external-%s-%s", actionType, hex.EncodeToString(sum[:16]))
}

func extractRuntimeAPIKey(r *stdhttp.Request) string {
	if key := strings.TrimSpace(r.Header.Get(externalAgentAPIKeyHeader)); key != "" {
		return key
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if token, found := strings.CutPrefix(authHeader, "Bearer "); found {
		return strings.TrimSpace(token)
	}
	return ""
}

func authStatus(err error) int {
	switch err.Error() {
	case "missing external agent api key", "invalid external agent api key", "external agent is not active", "api key does not match requested external agent":
		return stdhttp.StatusUnauthorized
	default:
		return stdhttp.StatusServiceUnavailable
	}
}

func issueAPIKey() (string, string, error) {
	token := randomID("anet")
	return token, hashAPIKey(token), nil
}

func hashAPIKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func buildWalletChallenge(agent domain.ExternalAgent) string {
	return fmt.Sprintf(
		"AetherNet External Agent Verification\nAgent ID: %s\nHandle: %s\nWallet: %s\nNonce: %s\nIssued At: %s\nPurpose: issue runtime API key",
		agent.ID,
		agent.Handle,
		agent.OwnerWalletAddress,
		randomID("nonce"),
		time.Now().UTC().Format(time.RFC3339),
	)
}

func randomID(prefix string) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return prefix + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(raw[:])
}

func normalizedHandle(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastDash = false
		case !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func nonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func agentKindFromID(agentID string) string {
	if strings.HasPrefix(agentID, "ext-") {
		return "external"
	}
	return "native"
}
