package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	// deps inyectadas luego (spotify.Service, store.Store, cryptor)
}

func NewAuthHandler() *AuthHandler { return &AuthHandler{} }

func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/start", h.Start)
	r.Get("/callback", h.Callback)
	r.Get("/status", h.Status)
	r.Post("/logout", h.Logout)
	r.Get("/token", h.Token)
}

// Start inicia flujo OAuth interactive (redirect).
// Por ahora devuelve URL de Spotify OAuth que el frontend debe abrir.
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	// TODO: generar state, crear sesión librespot en modo interactive y devolver URL
	writeJSON(w, http.StatusOK, map[string]any{
		"mode": "interactive",
		"url":  "http://localhost:8081/api/v1/spotify/auth/callback?code=mock_code&state=mock_state",
		"hint": "TODO: integrar session.NewSessionFromOptions con Credentials interactive + OAuth2 server en :36842",
	})
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	// TODO: intercambiar code por token vía ap.ConnectSpotifyToken + login5
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"code":   code,
		"state":  state,
		"next":   "persistir credenciales cifradas y redirigir a frontend",
	})
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	writeJSON(w, http.StatusOK, map[string]any{
		"userId": userID,
		"linked": false,
		"hint":   "TODO: consultar store.Store",
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	// Solo para debug interno, no exponer a frontend sin auth
	writeJSON(w, http.StatusOK, map[string]any{"token": "redacted - use Authorization: Bearer on proxy endpoints"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
