package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

type ProxyHandler struct {
	apiBase string // https://api.spotify.com
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{apiBase: "https://api.spotify.com"}
}

func (h *ProxyHandler) Routes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Get("/me/playlists", h.MePlaylists)
	r.Get("/playlists/{id}", h.Playlist)
	r.Get("/search", h.Search)
}

// Me proxy a api.spotify.com/v1/me usando token del usuario (header Authorization reenviado)
func (h *ProxyHandler) Me(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, "/v1/me", nil)
}

func (h *ProxyHandler) MePlaylists(w http.ResponseWriter, r *http.Request) {
	q := url.Values{}
	if v := r.URL.Query().Get("limit"); v != "" {
		q.Set("limit", v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		q.Set("offset", v)
	}
	h.forward(w, r, "/v1/me/playlists", q)
}

func (h *ProxyHandler) Playlist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.forward(w, r, "/v1/playlists/"+id, nil)
}

func (h *ProxyHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// q, type, limit, offset pasan directo a Spotify
	h.forward(w, r, "/v1/search", q)
}

func (h *ProxyHandler) forward(w http.ResponseWriter, r *http.Request, path string, q url.Values) {
	token := r.Header.Get("Authorization")
	if token == "" {
		// fallback a query ?access_token= para dev
		if t := r.URL.Query().Get("access_token"); t != "" {
			token = "Bearer " + t
		}
	}
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing Authorization Bearer token (login vía /auth/start primero)"})
		return
	}

	u := h.apiBase + path
	if q != nil && len(q) > 0 {
		u += "?" + q.Encode()
	} else if r.URL.RawQuery != "" && path == "/v1/search" {
		// ya viene en q
	}

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")

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

func writeJSONProxy(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
