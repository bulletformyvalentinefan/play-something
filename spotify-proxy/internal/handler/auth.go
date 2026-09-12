package handler

import (
	"context"
	"encoding/json"
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

// Start genera URL de OAuth redirect (ingeniería inversa: usa client_id oficial).
// Frontend debe abrir la URL en popup/nueva pestaña.
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.UserID == "" {
		req.UserID = r.URL.Query().Get("userId")
	}
	if req.UserID == "" {
		http.Error(w, `{"error":"userId requerido"}`, http.StatusBadRequest)
		return
	}
	cfg := auth.OAuthConfig(h.cfg.OAuthCallbackURL)
	state := auth.RandomState()

	h.mu.Lock()
	h.states[state] = req.UserID
	h.mu.Unlock()

	// El token se intercambia en /callback, luego se cifra y guarda.
	url := auth.AuthURLWithState(cfg, state)
	writeJSON(w, http.StatusOK, startResponse{URL: url, State: state})
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

	cfg := auth.OAuthConfig(h.cfg.OAuthCallbackURL)
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

	// Obtener perfil para spotifyUsername (opcional, best-effort)
	spotifyUsername := userID // fallback
	if tok.AccessToken != "" {
		if uname := fetchSpotifyUsername(r.Context(), tok); uname != "" {
			spotifyUsername = uname
		}
	}

	creds := store.Credentials{
		UserID:          userID,
		SpotifyUsername: spotifyUsername,
		DeviceID:        "play-something-" + userID[:min(8, len(userID))],
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		ExpiresAt:       tok.Expiry,
	}
	_ = h.st.Save(creds)

	h.mu.Lock()
	delete(h.states, state)
	h.mu.Unlock()

	// Redirigir a frontend con éxito (deep link para popup)
	frontend := h.cfg.FrontendOrigin
	if frontend == "" {
		frontend = "http://localhost:5173"
	}
	// El frontend debe cerrar popup y refrescar status
	http.Redirect(w, r, frontend+"/?spotify_linked=1&userId="+userID, http.StatusFound)
}

func fetchSpotifyUsername(ctx context.Context, tok *oauth2.Token) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me", nil)
	tok.SetAuthHeader(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.ID
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "userId requerido"})
		return
	}
	creds, err := h.st.Get(userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "linked": false})
		return
	}
	// Desencriptar solo para verificar expiración, no devolver token
	_ = creds
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":          userID,
		"linked":          true,
		"spotifyUsername": creds.SpotifyUsername,
		"expiresAt":       creds.ExpiresAt,
		"hasRefreshToken": creds.RefreshTokenEnc != "",
	})
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
		cfg := auth.OAuthConfig(h.cfg.OAuthCallbackURL)
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
