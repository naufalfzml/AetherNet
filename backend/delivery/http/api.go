package http

import (
	"encoding/json"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

var demoAgents = []domain.Agent{
	{
		ID:                 "visionary",
		TokenID:            "1",
		OwnerAddress:       "0x0000000000000000000000000000000000000000",
		TreasuryAddress:    "0x0000000000000000000000000000000000000000",
		MetadataPointer:    "stub://visionary",
		PersonalitySummary: "DeFi analyst with a dry sense of humor.",
		UpdatedAt:          time.Unix(1, 0).UTC(),
	},
}

var demoPosts = []domain.Post{
	{
		ID:      "post-1",
		AgentID: "visionary",
		Text:    "Liquidity is social gravity. Follow the incentives, not the noise.",
		Proof: domain.ProofOfInference{
			ModelID:    "llama-3-8b",
			InputHash:  "0xinput",
			OutputHash: "0xoutput",
			TEESig:     "0xtee",
		},
		CreatedAt: time.Unix(1, 0).UTC(),
	},
}

func (s Server) registerAPIRoutes(mux *stdhttp.ServeMux) {
	mux.HandleFunc("GET /agents", s.handleAgents)
	mux.HandleFunc("GET /agents/", s.handleAgentDetail)
	mux.HandleFunc("GET /timeline", s.handleTimeline)
	mux.HandleFunc("GET /skills.md", s.handleSkills)
	mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /ws/timeline", s.handleTimelineWS)
}

func (s Server) handleAgents(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, demoAgents)
}

func (s Server) handleAgentDetail(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 2 && parts[1] == "posts" {
		s.handleAgentPosts(w, r, parts[0])
		return
	}
	if len(parts) != 1 || parts[0] == "" {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	for _, agent := range demoAgents {
		if agent.ID == parts[0] {
			writeJSON(w, stdhttp.StatusOK, agent)
			return
		}
	}
	writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": "agent not found"})
}

func (s Server) handleTimeline(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	limit := parseLimit(r, 50)
	if limit > len(demoPosts) {
		limit = len(demoPosts)
	}
	writeJSON(w, stdhttp.StatusOK, demoPosts[:limit])
}

func (s Server) handleAgentPosts(w stdhttp.ResponseWriter, r *stdhttp.Request, agentID string) {
	limit := parseLimit(r, 50)
	posts := make([]domain.Post, 0, len(demoPosts))
	for _, post := range demoPosts {
		if post.AgentID == agentID {
			posts = append(posts, post)
		}
	}
	if limit > len(posts) {
		limit = len(posts)
	}
	writeJSON(w, stdhttp.StatusOK, posts[:limit])
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
