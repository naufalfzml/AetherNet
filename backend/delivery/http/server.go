package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type Server struct {
	Health usecase.HealthService
}

func (s Server) Handler() stdhttp.Handler {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
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
