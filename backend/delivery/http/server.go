package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type Server struct {
	Health  usecase.HealthService
	Metrics *usecase.Metrics
}

func (s Server) Handler() stdhttp.Handler {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	return mux
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
