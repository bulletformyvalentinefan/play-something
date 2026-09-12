package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type PlayerHandler struct{}

func NewPlayerHandler() *PlayerHandler { return &PlayerHandler{} }

func (h *PlayerHandler) Routes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Post("/play", h.Play)
	r.Post("/pause", h.Pause)
	r.Post("/next", h.Next)
	r.Post("/prev", h.Prev)
	r.Post("/seek", h.Seek)
	r.Post("/volume", h.Volume)
	r.Get("/events", h.Events)
}

func (h *PlayerHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "idle",
		"hint":   "TODO: conectar a go-librespot player + dealer. Requiere CGO libvorbis/flac/mpg123.",
	})
}

func (h *PlayerHandler) Play(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "player play TODO - integrar spclient + player"})
}
func (h *PlayerHandler) Pause(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "TODO"})
}
func (h *PlayerHandler) Next(w http.ResponseWriter, r *http.Request)  { writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "TODO"}) }
func (h *PlayerHandler) Prev(w http.ResponseWriter, r *http.Request)  { writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "TODO"}) }
func (h *PlayerHandler) Seek(w http.ResponseWriter, r *http.Request)  { writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "TODO"}) }
func (h *PlayerHandler) Volume(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "TODO"}) }
func (h *PlayerHandler) Events(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"hint": "TODO: WS /events proxy a dealer websocket"})
}
