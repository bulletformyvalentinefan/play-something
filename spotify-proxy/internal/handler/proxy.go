package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/config"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/crypto"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/store"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	spotifyAuth "github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/auth"
)

type ProxyHandler struct {
	cfg     config.Config
	st      store.Store
	cryptor *crypto.Cryptor
	apiBase string
}

func NewProxyHandler(cfg config.Config, st store.Store, c *crypto.Cryptor) *ProxyHandler {
	return &ProxyHandler{cfg: cfg, st: st, cryptor: c, apiBase: "https://api.spotify.com"}
}

func (h *ProxyHandler) Routes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Get("/me/playlists", h.MePlaylists)
	r.Get("/playlists/{id}", h.Playlist)
	r.Get("/search", h.Search)
	// compat con frontend viejo: /tracks/search
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
	h.forward(w, r, "/v1/search", q)
}

func (h *ProxyHandler) forward(w http.ResponseWriter, r *http.Request, path string, q url.Values) {
	token := h.resolveToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "no autorizado: vincula tu cuenta Spotify en /api/v1/spotify/auth/start",
			"hint":  "envia Authorization: Bearer <token> o ?userId=<uuid> si ya vinculaste",
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

	// Si Spotify devuelve 401, intentar refresh una vez si tenemos refresh_token
	if resp.StatusCode == http.StatusUnauthorized {
		if refreshed := h.tryRefresh(r.Context(), r); refreshed != "" {
			req2, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
			req2.Header.Set("Authorization", "Bearer "+refreshed)
			req2.Header.Set("Accept", "application/json")
			if resp2, err := http.DefaultClient.Do(req2); err == nil {
				defer resp2.Body.Close()
				w.Header().Set("Content-Type", resp2.Header.Get("Content-Type"))
				w.WriteHeader(resp2.StatusCode)
				_, _ = io.Copy(w, resp2.Body)
				return
			}
		}
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (h *ProxyHandler) resolveToken(r *http.Request) string {
	// 1) Authorization header
	if t := r.Header.Get("Authorization"); t != "" {
		// soporta "Bearer xxx" o solo xxx
		if len(t) > 7 && t[:7] == "Bearer " {
			return t[7:]
		}
		return t
	}
	// 2) ?access_token=
	if t := r.URL.Query().Get("access_token"); t != "" {
		return t
	}
	// 3) store por userId
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}
	if userID == "" {
		return ""
	}
	creds, err := h.st.Get(userID)
	if err != nil {
		return ""
	}
	tok, _ := h.cryptor.Decrypt(creds.AccessTokenEnc)
	// si expiró hace menos de 5 min, igual devolver (el forward hará refresh)
	return tok
}

func (h *ProxyHandler) tryRefresh(ctx context.Context, r *http.Request) string {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		userID = r.Header.Get("X-User-Id")
	}
	if userID == "" {
		return ""
	}
	creds, err := h.st.Get(userID)
	if err != nil || creds.RefreshTokenEnc == "" {
		return ""
	}
	refresh, _ := h.cryptor.Decrypt(creds.RefreshTokenEnc)
	cfg := spotifyAuth.OAuthConfig(h.cfg.OAuthCallbackURL)
	tok := &oauth2.Token{RefreshToken: refresh, Expiry: time.Now().Add(-time.Hour)}
	src := cfg.TokenSource(ctx, tok)
	newTok, err := src.Token()
	if err != nil {
		return ""
	}
	accessEnc, _ := h.cryptor.Encrypt(newTok.AccessToken)
	creds.AccessTokenEnc = accessEnc
	creds.ExpiresAt = newTok.Expiry
	if newTok.RefreshToken != "" {
		enc, _ := h.cryptor.Encrypt(newTok.RefreshToken)
		creds.RefreshTokenEnc = enc
	}
	_ = h.st.Save(creds)
	return newTok.AccessToken
}

func writeJSONProxy(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
