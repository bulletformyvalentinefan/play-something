package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	OracleDSN          string
	RedisAddr          string
	FrontendOrigin     string
	CredentialKey      string // 32 bytes hex or raw for AES-GCM
	SpotifyDeviceName  string
	SpotifyBitrate     int
	SpotifyCacheDir    string
	LogLevel           string
	OAuthCallbackURL   string
	SpotifyClientID    string
	SpotifyClientSecret string // solo si usas tu propia app con secret
}

func Load() Config {
	return Config{
		Port:             envOr("PORT", "8081"),
		OracleDSN:        envOr("ORACLE_DSN", "oracle://spotify_user:spotify_pwd@localhost:1521/FREEPDB1"),
		RedisAddr:        envOr("REDIS_ADDR", "localhost:6379"),
		FrontendOrigin:   envOr("FRONTEND_ORIGIN", "http://localhost:5173"),
		CredentialKey:    envOr("CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef"),
		SpotifyDeviceName: envOr("SPOTIFY_DEVICE_NAME", "play-something"),
		SpotifyBitrate:   envIntOr("SPOTIFY_BITRATE", 320),
		SpotifyCacheDir:  envOr("SPOTIFY_CACHE_DIR", ""),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		// IMPORTANTE: Spotify bloquea localhost desde 2025, debe ser 127.0.0.1
		// Si usas el client_id oficial 65b... SOLO está whitelisteado http://127.0.0.1:8989/login (ver Sonora crates/music/src/spotify/auth.rs:9)
		OAuthCallbackURL: envOr("OAUTH_CALLBACK_URL", "http://127.0.0.1:8081/api/v1/spotify/auth/callback"),
		SpotifyClientID:  envOr("SPOTIFY_CLIENT_ID", "65b708073fc0480ea92a077233ca87bd"),
		SpotifyClientSecret: envOr("SPOTIFY_CLIENT_SECRET", ""),
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envIntOr(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}
