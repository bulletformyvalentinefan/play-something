package main

import (
	"log"
	"net/http"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/config"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/crypto"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/handler"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/store"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func mustCryptor(key string) *crypto.Cryptor {
	c, err := crypto.NewCryptor(key)
	if err != nil {
		log.Fatalf("CREDENTIAL_KEY invalid (debe ser 32 bytes): %v", err)
	}
	return c
}

func main() {
	cfg := config.Load()
	cryptor := mustCryptor(cfg.CredentialKey)
	st := store.NewMemoryStore(cryptor)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendOrigin, "http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"spotify-proxy","mode":"go-librespot"}`))
	})

	authH := handler.NewAuthHandler(cfg, st, cryptor)
	proxyH := handler.NewProxyHandler(cfg, st, cryptor)
	playerH := handler.NewPlayerHandler()

	r.Route("/api/v1/spotify", func(r chi.Router) {
		r.Route("/auth", authH.Routes)
		r.Route("/proxy", proxyH.Routes)
		r.Route("/player", playerH.Routes)
	})

	// Compat: frontend actual usa /api/v1/spotify/tracks/search -> proxear luego a /proxy/search
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, "/health", http.StatusFound)
	})

	addr := ":" + cfg.Port
	log.Printf("spotify-proxy listening on %s (frontend origin %s)", addr, cfg.FrontendOrigin)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
