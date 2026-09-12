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
	playerH := handler.NewPlayerHandler(cfg, st, cryptor)

	r.Route("/api/v1/spotify", func(r chi.Router) {
		r.Route("/auth", authH.Routes)
		r.Route("/proxy", proxyH.Routes)
		r.Route("/player", playerH.Routes)
	})

	// Loopback handler para client_id oficial (Sonora style http://127.0.0.1:8989/login)
	// Spotify solo whitelistea ese redirect para el client 65b...; lo exponemos además del callback en :8081
	r.Get("/login", authH.Callback)

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, "/health", http.StatusFound)
	})

	// Si usamos client oficial, levantar también listener en :8989 (whitelisted por Spotify)
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientID == "65b708073fc0480ea92a077233ca87bd" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/login", authH.Callback)
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`{"status":"ok","loopback":":8989"}`))
			})
			log.Printf("loopback OAuth listening on :8989 (for official client_id %s)", cfg.SpotifyClientID)
			if err := http.ListenAndServe(":8989", mux); err != nil {
				log.Printf("loopback :8989 error: %v", err)
			}
		}()
	}

	addr := ":" + cfg.Port
	log.Printf("spotify-proxy listening on %s (frontend origin %s) client_id=%s callback=%s", addr, cfg.FrontendOrigin, cfg.SpotifyClientID, cfg.OAuthCallbackURL)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
