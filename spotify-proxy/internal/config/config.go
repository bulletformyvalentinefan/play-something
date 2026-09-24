package config

import (
	"os"
)

type Config struct {
	Port             string
	FrontendOrigin   string
	OAuthCallbackURL string
	SpotifyClientID  string
}

func Load() Config {
	return Config{
		Port:           envOr("PORT", "8081"),
		FrontendOrigin: envOr("FRONTEND_ORIGIN", "http://localhost:5173"),
		// IMPORTANTE: Spotify bloquea localhost desde 2025, debe ser 127.0.0.1
		// Con el client_id oficial SOLO vale http://127.0.0.1:8989/login
		OAuthCallbackURL: envOr("OAUTH_CALLBACK_URL", "http://127.0.0.1:8081/api/v1/spotify/auth/callback"),
		SpotifyClientID:  envOr("SPOTIFY_CLIENT_ID", "65b708073fc0480ea92a077233ca87bd"),
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
