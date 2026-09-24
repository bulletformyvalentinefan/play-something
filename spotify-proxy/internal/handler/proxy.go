package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/spotify"
)

// ProxyHandler reenvía a Web API con el token del usuario; la búsqueda va por spclient.
type ProxyHandler struct {
	apiBase string
	mgr     *spotify.Manager
}

func NewProxyHandlerWithManager(mgr *spotify.Manager) *ProxyHandler {
	return &ProxyHandler{apiBase: "https://api.spotify.com", mgr: mgr}
}

func (h *ProxyHandler) Routes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Get("/me/playlists", h.MePlaylists)
	r.Post("/me/playlists", h.CreatePlaylist)
	r.Get("/playlists/{id}", h.Playlist)
	r.Put("/playlists/{id}", h.UpdatePlaylist)
	r.Delete("/playlists/{id}", h.DeletePlaylist)
	r.Get("/playlists/{id}/tracks", h.PlaylistTracks)
	r.Post("/playlists/{id}/tracks", h.AddTrack)
	r.Delete("/playlists/{id}/tracks", h.RemoveTrack)
	r.Get("/search", h.Search)
	r.Get("/tracks/search", h.Search)
	r.Get("/tracks/{id}", h.Track)
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

func (h *ProxyHandler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	// POST /v1/me/playlists no existe en Spotify — necesitamos /v1/users/{user_id}/playlists
	// Obtenemos user_id via /v1/me con el mismo token
	token := resolveBearer(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "no autorizado"})
		return
	}
	meReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, h.apiBase+"/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meResp, err := http.DefaultClient.Do(meReq)
	if err != nil || meResp.StatusCode != 200 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "no se pudo obtener perfil para crear playlist"})
		return
	}
	defer meResp.Body.Close()
	var me struct{ ID string `json:"id"` }
	_ = json.NewDecoder(meResp.Body).Decode(&me)
	body, _ := io.ReadAll(r.Body)
	h.forwardWithBody(w, r, "/v1/users/"+me.ID+"/playlists", nil, body)
}

func (h *ProxyHandler) Playlist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.forward(w, r, "/v1/playlists/"+id, nil)
}

func (h *ProxyHandler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, _ := io.ReadAll(r.Body)
	h.forwardWithBody(w, r, "/v1/playlists/"+id, nil, body)
}

func (h *ProxyHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Spotify usa DELETE /v1/playlists/{id}/followers para unfollow
	h.forwardWithMethod(w, r, "/v1/playlists/"+id+"/followers", nil, nil, http.MethodDelete)
}

func (h *ProxyHandler) PlaylistTracks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q := r.URL.Query()
	q.Del("userId")
	h.forward(w, r, "/v1/playlists/"+id+"/tracks", q)
}

func (h *ProxyHandler) AddTrack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, _ := io.ReadAll(r.Body)
	// Spotify espera { uris: ["spotify:track:..."] } y opcional position
	h.forwardWithBody(w, r, "/v1/playlists/"+id+"/tracks", nil, body)
}

func (h *ProxyHandler) RemoveTrack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, _ := io.ReadAll(r.Body)
	h.forwardWithBodyAndMethod(w, r, "/v1/playlists/"+id+"/tracks", nil, body, http.MethodDelete)
}

func (h *ProxyHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Del("userId")
	q.Del("access_token")
	// Sonora: spclient.ContextResolve("spotify:search:"+escaped) — primario, sin fallback Web API
	if h.mgr != nil {
		if token := resolveBearer(r); token != "" {
			limit := spotify.DefaultSearchLimit
			if v := q.Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					limit = n
				}
			}
			if results, err := h.mgr.SearchWithToken(r.Context(), token, q.Get("q"), limit); err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Source", "spclient")
				items := make([]map[string]any, 0, len(results))
				for _, t := range results {
				items = append(items, map[string]any{
					"id": t.ID, "name": t.Name, "uri": t.URI, "duration_ms": t.DurationMs, "preview_url": nil,
					"artists": []map[string]string{{"name": t.Artist}},
					"album": map[string]any{"name": t.Album, "images": []map[string]string{{"url": t.AlbumCover}}},
				})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"tracks": map[string]any{"items": items}})
				return
			}
		}
	}
	// Sin resultados: aún devolvemos 200 con X-Source: spclient para que el frontend no intente re-req
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"tracks": map[string]any{"items": []any{}}})
}

func (h *ProxyHandler) Track(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.forward(w, r, "/v1/tracks/"+id, nil)
}

func (h *ProxyHandler) forward(w http.ResponseWriter, r *http.Request, path string, q url.Values) {
	h.forwardWithMethod(w, r, path, q, nil, http.MethodGet)
}

func (h *ProxyHandler) forwardWithBody(w http.ResponseWriter, r *http.Request, path string, q url.Values, body []byte) {
	h.forwardWithBodyAndMethod(w, r, path, q, body, http.MethodPost)
}

func (h *ProxyHandler) forwardWithBodyAndMethod(w http.ResponseWriter, r *http.Request, path string, q url.Values, body []byte, method string) {
	h.forwardWithMethod(w, r, path, q, body, method)
}

func (h *ProxyHandler) forwardWithMethod(w http.ResponseWriter, r *http.Request, path string, q url.Values, body []byte, method string) {
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
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(r.Context(), method, u, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Sonora usa spclient que no tiene este límite; Web API sí — respetamos Retry-After
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			w.Header().Set("Retry-After", ra)
			if secs, err := strconv.Atoi(ra); err == nil {
				w.Header().Set("X-Retry-In", strconv.Itoa(secs))
			}
		}
	}
	b, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		w.Header().Set("Retry-After", ra)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(b)
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
	return ""
}
