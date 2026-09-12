package main

import (
	"log"
	"net/http"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/config"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/handler"
	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/spotify"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendOrigin, "http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"spotify-proxy","mode":"sonora-go-librespot"}`))
	})

	// Sonora-style: Session go-librespot (client_id 65b... solo handshake, todo via spclient)
	mgr := spotify.NewManager()
	_ = mgr // placeholder para evitar unused si no se usa aún en stub Windows
	authH := handler.NewAuthHandler(cfg, mgr)
	proxyH := handler.NewProxyHandlerWithManager(mgr)
	playerH := handler.NewPlayerHandler()

	r.Route("/api/v1/spotify", func(r chi.Router) {
		r.Route("/auth", authH.Routes)
		r.Route("/proxy", proxyH.Routes)
		r.Route("/player", playerH.Routes)
	})

	// Loopback whitelisted para client oficial 65b... (Sonora auth.rs:12)
	r.Get("/login", authH.Callback)

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, "/health", http.StatusFound)
	})

	if cfg.SpotifyClientID == "" || cfg.SpotifyClientID == "65b708073fc0480ea92a077233ca87bd" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/login", authH.Callback)
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`{"status":"ok","loopback":":8989"}`))
			})
			log.Printf("loopback OAuth listening on :8989 (official client_id)")
			if err := http.ListenAndServe(":8989", mux); err != nil {
				log.Printf("loopback :8989 error: %v", err)
			}
		}()
	}

	addr := ":" + cfg.Port
	log.Printf("spotify-proxy listening on %s frontend=%s client_id=%s", addr, cfg.FrontendOrigin, cfg.SpotifyClientID)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
