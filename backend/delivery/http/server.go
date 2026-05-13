package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type Server struct {
	Health         usecase.HealthService
	Metrics        *usecase.Metrics
	Config         config.Config
	Agents         usecase.AgentRepository
	ExternalAgents usecase.ExternalAgentRepository
	Events         usecase.SocialEventRepository
	Metadata       usecase.AgentMetadataRepository
	Compute        usecase.ZGComputeClient
	Storage        usecase.ZGStorageClient
}

func (s Server) Handler() stdhttp.Handler {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	s.registerAPIRoutes(mux)
	return withCORS(mux)
}

func withCORS(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Aethernet-Agent-Key")
		if r.Method == stdhttp.MethodOptions {
			w.WriteHeader(stdhttp.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s Server) handleHealth(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	report := s.Health.Check(r.Context())
	if report.Status != "ok" {
		w.WriteHeader(stdhttp.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(report)
}

func (s Server) handleMetrics(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if s.Metrics == nil {
		_ = json.NewEncoder(w).Encode(map[string]uint64{"cycle_count": 0, "failures": 0})
		return
	}
	_ = json.NewEncoder(w).Encode(s.Metrics.Snapshot())
}
