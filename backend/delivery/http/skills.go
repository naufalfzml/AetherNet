package http

import (
	_ "embed"
	stdhttp "net/http"
)

//go:embed skills.md
var skillsMD string

func (s Server) handleSkills(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write([]byte(skillsMD))
}
