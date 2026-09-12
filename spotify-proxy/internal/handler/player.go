package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

type PlayerHandler struct{}

func NewPlayerHandler() *PlayerHandler { return &PlayerHandler{} }

func (h *PlayerHandler) Routes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Put("/play", h.Play)
	r.Put("/pause", h.Pause)
	r.Post("/next", h.Next)
	r.Post("/prev", h.Prev)
	r.Put("/seek", h.Seek)
	r.Put("/volume", h.Volume)
	r.Get("/events", h.Events)
	// compat POST
	r.Post("/play", h.Play)
	r.Post("/pause", h.Pause)
	r.Post("/seek", h.Seek)
	r.Post("/volume", h.Volume)
}

func (h *PlayerHandler) Status(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	h.proxyPlayer(w, r, http.MethodGet, "https://api.spotify.com/v1/me/player", nil, token)
}

func (h *PlayerHandler) Play(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	body, _ := io.ReadAll(r.Body)
	deviceID := r.URL.Query().Get("device_id")
	u := "https://api.spotify.com/v1/me/player/play"
	if deviceID != "" {
		u += "?device_id=" + url.QueryEscape(deviceID)
	}
	h.proxyPlayer(w, r, http.MethodPut, u, body, token)
}

func (h *PlayerHandler) Pause(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	u := "https://api.spotify.com/v1/me/player/pause"
	if deviceID != "" {
		u += "?device_id=" + url.QueryEscape(deviceID)
	}
	h.proxyPlayer(w, r, http.MethodPut, u, nil, token)
}

func (h *PlayerHandler) Next(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	h.proxyPlayer(w, r, http.MethodPost, "https://api.spotify.com/v1/me/player/next", nil, token)
}

func (h *PlayerHandler) Prev(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	h.proxyPlayer(w, r, http.MethodPost, "https://api.spotify.com/v1/me/player/previous", nil, token)
}

func (h *PlayerHandler) Seek(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	pos := r.URL.Query().Get("position_ms")
	if pos == "" {
		pos = "0"
	}
	u := "https://api.spotify.com/v1/me/player/seek?position_ms=" + url.QueryEscape(pos)
	h.proxyPlayer(w, r, http.MethodPut, u, nil, token)
}

func (h *PlayerHandler) Volume(w http.ResponseWriter, r *http.Request) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no vinculado"})
		return
	}
	vol := r.URL.Query().Get("volume_percent")
	if vol == "" {
		vol = "50"
	}
	u := "https://api.spotify.com/v1/me/player/volume?volume_percent=" + url.QueryEscape(vol)
	h.proxyPlayer(w, r, http.MethodPut, u, nil, token)
}

func (h *PlayerHandler) Events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte("event: hint\ndata: spclient dealer WS en linux via librespot\n\n"))
}

func (h *PlayerHandler) proxyPlayer(w http.ResponseWriter, r *http.Request, method, target string, body []byte, token string) {
	req, _ := http.NewRequestWithContext(r.Context(), method, target, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if len(body) == 0 {
		req.Header.Del("Content-Type")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
