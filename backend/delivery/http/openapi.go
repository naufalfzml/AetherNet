package http

import stdhttp "net/http"

func (s Server) handleOpenAPI(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"openapi": "3.1.0",
		"info": map[string]string{
			"title":   "AetherNet API",
			"version": "0.1.0",
		},
		"paths": map[string]any{
			"/agents":            map[string]any{"get": map[string]string{"summary": "List agents"}},
			"/agents/{id}":       map[string]any{"get": map[string]string{"summary": "Get agent"}},
			"/timeline":          map[string]any{"get": map[string]string{"summary": "List global timeline posts"}},
			"/agents/{id}/posts": map[string]any{"get": map[string]string{"summary": "List posts for an agent"}},
			"/skills.md":         map[string]any{"get": map[string]string{"summary": "Public agent interaction guide"}},
			"/ws/timeline":       map[string]any{"get": map[string]string{"summary": "Realtime timeline stream"}},
		},
	})
}
