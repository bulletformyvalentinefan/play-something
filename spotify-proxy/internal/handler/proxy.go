package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// ProxyHandler Sonora-style: sin BDD, solo forward con Authorization: Bearer.
// El frontend manda el token que obtuvo en /auth, el Go lo pasa directo a api.spotify.com.
// Esto es lo que hace Sonora con session.spclient() pero sin guardar nada en disco.
type ProxyHandler struct {
	apiBase string
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{apiBase: "https://api.spotify.com"}
}

func (h *ProxyHandler) Routes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Get("/me/playlists", h.MePlaylists)
	r.Get("/playlists/{id}", h.Playlist)
	r.Get("/search", h.Search)
	r.Get("/tracks/search", h.Search)
}

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
	// filtra userId/access_token que no son de Spotify
	q.Del("userId")
	q.Del("access_token")
	h.forward(w, r, "/v1/search", q)
}

func (h *ProxyHandler) forward(w http.ResponseWriter, r *http.Request, path string, q url.Values) {
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "no autorizado: hace login con Spotify primero",
			"hint":  "POST /api/v1/spotify/auth/start -> redirect -> el token va en Authorization: Bearer",
		})
		return
	}
	u := h.apiBase + path
	if q != nil && len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

func resolveBearer(r *http.Request) string {
	if t := r.Header.Get("Authorization"); t != "" {
		if len(t) > 7 && t[:7] == "Bearer " {
			return t[7:]
		}
		return t
	}
	if t := r.URL.Query().Get("access_token"); t != "" {
		return t
	}
	// compat: frontend viejo mandaba ?userId, pero ya no guardamos BDD
	// si solo hay userId sin token, no podemos resolver -> pide header
	return ""
}

func writeJSONProxy(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
