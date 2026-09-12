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
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// AuthHandler Sonora-style: sin BDD, sin AES, solo memoria + spclient.
// El token vive en memoria (como Session en Sonora auth.rs:112) y
// el frontend lo recibe y lo manda en Authorization en cada request.
type AuthHandler struct {
	cfg config.Config
	mu  sync.Mutex
	states map[string]string
	// PKCE verifier per state (Sonora: PkceCodeChallenge::new_random_sha256)
	verifiers map[string]string
	// userID (spotify id) -> token
	tokens   map[string]*oauth2.Token
	profiles map[string]spotifyProfile
}

func NewAuthHandler(cfg config.Config) *AuthHandler {
	return &AuthHandler{
		cfg:       cfg,
		states:    make(map[string]string),
		verifiers: make(map[string]string),
		tokens:    make(map[string]*oauth2.Token),
		profiles:  make(map[string]spotifyProfile),
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
	// Sonora: DEFAULT_REDIRECT_URI=http://127.0.0.1:8989/login (auth.rs:12)
	// Nosotros: si client oficial y callback es 8081/callback -> forzar 8989/login
	redirect := h.cfg.OAuthCallbackURL
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
		redirect = auth.OfficialRedirectURI
	}
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
	redirect := h.cfg.OAuthCallbackURL
	if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
		redirect = auth.OfficialRedirectURI
	}
	cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
	tok, err := cfg.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, `{"error":"token exchange failed: `+err.Error()+`"}`, http.StatusBadGateway)
		return
	}
	profile := fetchSpotifyProfile(r.Context(), tok)
	spotifyID := profile.ID
	if spotifyID == "" {
		spotifyID = "spotify-user"
	}
	h.mu.Lock()
	h.tokens[spotifyID] = tok
	h.profiles[spotifyID] = profile
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
	h.mu.Lock()
	defer h.mu.Unlock()
	if userID == "" {
		// sin userId devuelve el primero vinculado (Sonora web)
		for id, tok := range h.tokens {
			p := h.profiles[id]
			writeJSON(w, http.StatusOK, map[string]any{
				"userId": id, "linked": true, "spotifyUsername": p.ID, "display_name": p.DisplayName, "expiresAt": tok.Expiry,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"linked": false})
		return
	}
	tok, ok := h.tokens[userID]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "linked": false})
		return
	}
	p := h.profiles[userID]
	writeJSON(w, http.StatusOK, map[string]any{
		"userId": userID, "linked": true, "spotifyUsername": p.ID, "display_name": p.DisplayName, "expiresAt": tok.Expiry,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	h.mu.Lock()
	var tok *oauth2.Token
	if userID == "" {
		for _, t := range h.tokens {
			tok = t
			break
		}
	} else {
		tok = h.tokens[userID]
	}
	h.mu.Unlock()
	if tok == nil {
		http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
		return
	}
	// proxy fresco a Spotify con el token en memoria
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
	h.mu.Lock()
	if userID == "" {
		// borra todo (un usuario)
		h.tokens = make(map[string]*oauth2.Token)
		h.profiles = make(map[string]spotifyProfile)
	} else {
		delete(h.tokens, userID)
		delete(h.profiles, userID)
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	h.mu.Lock()
	var tok *oauth2.Token
	if userID == "" {
		for _, t := range h.tokens {
			tok = t
			break
		}
	} else {
		tok = h.tokens[userID]
	}
	h.mu.Unlock()
	if tok == nil {
		http.Error(w, `{"error":"not linked"}`, http.StatusNotFound)
		return
	}
	// refresh si expiró (Sonora lo hace via Session::connect con cache)
	if time.Now().After(tok.Expiry.Add(-30*time.Second)) && tok.RefreshToken != "" {
		redirect := h.cfg.OAuthCallbackURL
		if auth.IsOfficialClient(h.cfg.SpotifyClientID) && redirect == "http://127.0.0.1:8081/api/v1/spotify/auth/callback" {
			redirect = auth.OfficialRedirectURI
		}
		cfg := auth.OAuthConfig(redirect, h.cfg.SpotifyClientID)
		src := cfg.TokenSource(context.Background(), tok)
		if newTok, err := src.Token(); err == nil {
			h.mu.Lock()
			if userID == "" {
				for id := range h.tokens {
					h.tokens[id] = newTok
					tok = newTok
					break
				}
			} else {
				h.tokens[userID] = newTok
				tok = newTok
			}
			h.mu.Unlock()
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
