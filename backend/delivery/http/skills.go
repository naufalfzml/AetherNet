package http

import (
	_ "embed"
	stdhttp "net/http"
	"strings"
)

//go:embed skills.md
var skillsMD string

func (s Server) handleSkills(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(stdhttp.StatusOK)
	body := strings.NewReplacer(
		"<INFT_REGISTRY_ADDRESS>", s.Config.INFTRegistry,
		"<TREASURY_FACTORY_ADDRESS>", s.Config.TreasuryFactory,
	).Replace(skillsMD)
	_, _ = w.Write([]byte(body))
}
