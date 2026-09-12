package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/oauth2"
)

// Spotify OAuth usando client_id oficial de go-librespot (ingeniería inversa).
// Este client_id es el que usa la app oficial de escritorio.
// Scopes necesarios para leer playlists, perfil y streaming.
const (
	SpotifyClientID = "65b708073fc0480ea92a077233ca87bd"
	AuthURL         = "https://accounts.spotify.com/authorize"
	TokenURL        = "https://accounts.spotify.com/api/token"
)

var Scopes = []string{
	"user-read-private",
	"user-read-email",
	"playlist-read-private",
	"playlist-read-collaborative",
	"user-library-read",
	"streaming",
	"user-modify-playback-state",
	"user-read-playback-state",
}

func OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    SpotifyClientID,
		Endpoint:    oauth2.Endpoint{AuthURL: AuthURL, TokenURL: TokenURL},
		RedirectURL: redirectURL,
		Scopes:      Scopes,
	}
}

func RandomState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func AuthURLWithState(cfg *oauth2.Config, state string) string {
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}
