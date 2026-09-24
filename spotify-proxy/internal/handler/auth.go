package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/auth"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/config"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/spotify"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// AuthHandler: OAuth PKCE contra accounts.spotify.com. El token se usa para
// crear la sesión spclient de go-librespot, no para Web API (salvo perfil/player).
type AuthHandler struct {
	cfg       config.Config
	mgr       *spotify.Manager
	mu        sync.Mutex
	states    map[string]string
	verifiers map[string]string
}

// redirectURI resuelve el redirect a usar: con el client oficial Spotify solo
// acepta el loopback whitelisteado, salvo callback propio configurado.
func (h *AuthHandler) redirectURI() string {
	redirect := h.cfg.OAuthCallbackURL
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
		redirect = auth.OfficialRedirectURI
	}
	return redirect
}

func NewAuthHandler(cfg config.Config, mgr *spotify.Manager) *AuthHandler {
	return &AuthHandler{
		cfg:       cfg,
		mgr:       mgr,
		states:    make(map[string]string),
		verifiers: make(map[string]string),
	}
}

func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/start", h.Start)
	r.Get("/callback", h.Callback)
	r.Get("/status", h.Status)
	r.Get("/me", h.Me)
	r.Post("/logout", h.Logout)
	r.Get("/token", h.Token)
}

func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	redirect := h.redirectURI()
	cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
	state := auth.RandomState()
	verifier := auth.RandomVerifier()

	h.mu.Lock()
	h.states[state] = "anon"
	h.verifiers[state] = verifier
	h.mu.Unlock()

	url := auth.AuthURLWithPKCE(cfg, state, verifier)
	hint := ""
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect != auth.OfficialRedirectURI {
		hint = "client_id oficial solo whitelistea " + auth.OfficialRedirectURI + " - crea app en developer.spotify.com o usa SPOTIFY_CLIENT_ID propio"
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
	_, ok := h.states[state]
	verifier := h.verifiers[state]
	h.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}
	cfg := auth.OAuthConfig(h.redirectURI(), h.cfg.SpotifyClientID)
	tok, err := cfg.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		h.mu.Lock()
		delete(h.states, state)
		delete(h.verifiers, state)
		h.mu.Unlock()
		http.Error(w, `{"error":"token exchange failed: `+err.Error()+`"}`, http.StatusBadGateway)
		return
	}
	profile := fetchSpotifyProfile(r.Context(), tok)
	spotifyID := profile.ID
	if spotifyID == "" {
		spotifyID = "spotify-user"
	}
	h.mgr.Save(spotifyID, tok, spotify.Profile{ID: profile.ID, DisplayName: profile.DisplayName, Email: profile.Email, Image: ""})
	h.mgr.Warmup(spotifyID)
	h.mu.Lock()
	delete(h.states, state)
	delete(h.verifiers, state)
	h.mu.Unlock()

	frontend := h.cfg.FrontendOrigin
	if frontend == "" {
		frontend = "http://localhost:5173"
	}
	// Incluimos token en redirect para persistir inmediato sin fetch extra
	http.Redirect(w, r, frontend+"/?spotify_linked=1&userId="+url.QueryEscape(spotifyID)+"&display_name="+url.QueryEscape(profile.DisplayName)+"&access_token="+url.QueryEscape(tok.AccessToken), http.StatusFound)
}

type spotifyProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Images      []struct {
		URL string `json:"url"`
	} `json:"images"`
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

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		if id, tok, ok := h.mgr.First(); ok {
			p, _ := h.mgr.GetProfile(id)
			writeJSON(w, http.StatusOK, map[string]any{
				"userId": id, "linked": true, "spotifyUsername": p.ID, "display_name": p.DisplayName, "expiresAt": tok.Expiry,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"linked": false})
		return
	}
	tok, ok := h.mgr.GetToken(userID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "linked": false})
		return
	}
	p, _ := h.mgr.GetProfile(userID)
	writeJSON(w, http.StatusOK, map[string]any{
		"userId": userID, "linked": true, "spotifyUsername": p.ID, "display_name": p.DisplayName, "expiresAt": tok.Expiry,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	var tok *oauth2.Token
	if userID == "" {
		_, tok, _ = h.mgr.First()
	} else {
		tok, _ = h.mgr.GetToken(userID)
	}
	if tok == nil {
		http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", "https://api.spotify.com/v1/me", nil)
	tok.SetAuthHeader(req)
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
		h.mgr.Clear()
	} else {
		h.mgr.Delete(userID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	var tok *oauth2.Token
	var id string
	if userID == "" {
		id, tok, _ = h.mgr.First()
		userID = id
	} else {
		tok, _ = h.mgr.GetToken(userID)
	}
	if tok == nil {
		http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
		return
	}
	if time.Now().After(tok.Expiry.Add(-30*time.Second)) && tok.RefreshToken != "" {
		cfg := auth.OAuthConfig(h.redirectURI(), h.cfg.SpotifyClientID)
		src := cfg.TokenSource(context.Background(), tok)
		if newTok, err := src.Token(); err == nil {
			p, _ := h.mgr.GetProfile(userID)
			h.mgr.Save(userID, newTok, p)
			tok = newTok
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok.AccessToken,
		"expires_at":   tok.Expiry,
		"token_type":   "Bearer",
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
