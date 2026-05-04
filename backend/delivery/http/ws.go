package http

import (
	"fmt"
	stdhttp "net/http"
)

func (s Server) handleTimelineWS(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		writeJSON(w, stdhttp.StatusUpgradeRequired, map[string]string{"error": "websocket upgrade required"})
		return
	}
	w.Header().Set("Sec-WebSocket-Version", "13")
	w.WriteHeader(stdhttp.StatusSwitchingProtocols)
	_, _ = fmt.Fprintln(w)
}
