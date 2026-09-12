package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/auth"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/config"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/crypto"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/store"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	cfg     config.Config
	st      store.Store
	cryptor *crypto.Cryptor
	// state -> userID (anti-CSRF). En prod usar Redis.
	mu     sync.Mutex
	states map[string]string
}

func NewAuthHandler(cfg config.Config, st store.Store, c *crypto.Cryptor) *AuthHandler {
	return &AuthHandler{cfg: cfg, st: st, cryptor: c, states: make(map[string]string)}
}

func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/start", h.Start)
	r.Get("/callback", h.Callback)
	r.Get("/status", h.Status)
	r.Get("/me", h.Me)
	r.Post("/logout", h.Logout)
	r.Get("/token", h.Token)
}

type startRequest struct {
	UserID string `json:"userId"`
}

type startResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

// Start genera URL de OAuth redirect.
// Si SPOTIFY_CLIENT_ID es el oficial 65b... SOLO funciona con http://127.0.0.1:8989/login (ver Sonora).
// Si creas tu app en developer.spotify.com, usa esa URI whitelisteada en OAUTH_CALLBACK_URL (debe ser 127.0.0.1, no localhost).
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.UserID == "" {
		req.UserID = r.URL.Query().Get("userId")
	}
	anonID := req.UserID
	if anonID == "" {
		anonID = "anon"
	}
	// elegir redirect según client_id: oficial -> 8989/login, custom -> el configurado
	redirect := h.cfg.OAuthCallbackURL
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
		// forzar whitelisteado de Sonora/librespot si quedó el default viejo con 8081
		redirect = auth.OfficialRedirectURI
	}
	cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
	state := auth.RandomState()

	h.mu.Lock()
	h.states[state] = anonID
	h.mu.Unlock()

	url := auth.AuthURLWithState(cfg, state)
	// hint para debug si es oficial y redirect no whitelisteado
	hint := ""
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect != auth.OfficialRedirectURI {
		hint = "ADVERTENCIA: client_id oficial solo whitelistea " + auth.OfficialRedirectURI + " - crea tu app en developer.spotify.com o usa SPOTIFY_CLIENT_ID propio y registra " + redirect + " (debe ser 127.0.0.1, no localhost)"
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": url, "state": state, "redirect_uri": redirect, "hint": hint})
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		http.Error(w, `{"error":"`+errParam+`"}`, http.StatusBadRequest)
		return
	}
	if code == "" || state == "" {
		http.Error(w, `{"error":"missing code or state"}`, http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	userID, ok := h.states[state]
	h.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}

	redirect := h.cfg.OAuthCallbackURL
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
		redirect = auth.OfficialRedirectURI
	}
	cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
	// Intercambiar code por token contra accounts.spotify.com
	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, `{"error":"token exchange failed: `+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	// Cifrar tokens antes de persistir (AES-GCM base64)
	accessEnc, _ := h.cryptor.Encrypt(tok.AccessToken)
	refreshEnc := ""
	if tok.RefreshToken != "" {
		refreshEnc, _ = h.cryptor.Encrypt(tok.RefreshToken)
	}

	// Obtener perfil real de Spotify (id, display_name, email)
	profile := fetchSpotifyProfile(r.Context(), tok)
	spotifyUsername := profile.ID
	if spotifyUsername == "" {
		spotifyUsername = userID
	}
	// El UserID estable pasa a ser el spotify id (si es anon) o el que vino
	realUserID := userID
	if userID == "anon" || userID == "" {
		realUserID = spotifyUsername
	}

	creds := store.Credentials{
		UserID:          realUserID,
		SpotifyUsername: spotifyUsername,
		DeviceID:        "play-something-" + realUserID[:min(8, len(realUserID))],
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		ExpiresAt:       tok.Expiry,
	}
	_ = h.st.Save(creds)
	// también guardar bajo anon para compat, pero principal es realUserID

	h.mu.Lock()
	delete(h.states, state)
	h.mu.Unlock()

	frontend := h.cfg.FrontendOrigin
	if frontend == "" {
		frontend = "http://localhost:5173"
	}
	http.Redirect(w, r, frontend+"/?spotify_linked=1&userId="+realUserID, http.StatusFound)
}

type spotifyProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func fetchSpotifyProfile(ctx context.Context, tok *oauth2.Token) spotifyProfile {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me", nil)
	tok.SetAuthHeader(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return spotifyProfile{}
	}
	defer resp.Body.Close()
	var p spotifyProfile
	_ = json.NewDecoder(resp.Body).Decode(&p)
	return p
}

func fetchSpotifyUsername(ctx context.Context, tok *oauth2.Token) string {
	return fetchSpotifyProfile(ctx, tok).ID
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		// sin userId intenta devolver el primero vinculado (útil para flujo Spotify-only)
		list, _ := h.st.List()
		if len(list) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"linked": false})
			return
		}
		creds := list[0]
		writeJSON(w, http.StatusOK, map[string]any{
			"userId":          creds.UserID,
			"linked":          true,
			"spotifyUsername": creds.SpotifyUsername,
			"expiresAt":       creds.ExpiresAt,
			"hasRefreshToken": creds.RefreshTokenEnc != "",
		})
		return
	}
	creds, err := h.st.Get(userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "linked": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":          userID,
		"linked":          true,
		"spotifyUsername": creds.SpotifyUsername,
		"expiresAt":       creds.ExpiresAt,
		"hasRefreshToken": creds.RefreshTokenEnc != "",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	var creds store.Credentials
	var err error
	if userID == "" {
		list, _ := h.st.List()
		if len(list) == 0 {
			http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
			return
		}
		creds = list[0]
	} else {
		creds, err = h.st.Get(userID)
		if err != nil {
			http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
			return
		}
	}
	access, _ := h.cryptor.Decrypt(creds.AccessTokenEnc)
	req, _ := http.NewRequestWithContext(r.Context(), "GET", "https://api.spotify.com/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		_ = json.NewDecoder(r.Body).Decode(&struct{ UserID string `json:"userId"` }{})
	}
	if userID != "" {
		_ = h.st.Delete(userID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, `{"error":"userId requerido"}`, http.StatusBadRequest)
		return
	}
	creds, err := h.st.Get(userID)
	if err != nil {
		http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
		return
	}
	// Renovar si expiró y hay refresh_token
	if time.Now().After(creds.ExpiresAt.Add(-30 * time.Second)) && creds.RefreshTokenEnc != "" {
		refresh, _ := h.cryptor.Decrypt(creds.RefreshTokenEnc)
		redirect := h.cfg.OAuthCallbackURL
		if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
			redirect = auth.OfficialRedirectURI
		}
		cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
		tok := &oauth2.Token{RefreshToken: refresh}
		src := cfg.TokenSource(context.Background(), tok)
		newTok, err := src.Token()
		if err == nil {
			accessEnc, _ := h.cryptor.Encrypt(newTok.AccessToken)
			creds.AccessTokenEnc = accessEnc
			creds.ExpiresAt = newTok.Expiry
			if newTok.RefreshToken != "" {
				enc, _ := h.cryptor.Encrypt(newTok.RefreshToken)
				creds.RefreshTokenEnc = enc
			}
			_ = h.st.Save(creds)
		}
	}
	access, _ := h.cryptor.Decrypt(creds.AccessTokenEnc)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"expires_at":   creds.ExpiresAt,
		"token_type":   "Bearer",
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
