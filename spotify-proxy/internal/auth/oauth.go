package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/oauth2"
)

// Spotify OAuth - soporta tanto el client_id oficial de go-librespot (ingeniería inversa)
// como tu propia app de https://developer.spotify.com/dashboard
// Ver Sonora: crates/music/src/spotify/auth.rs:9 DEFAULT_REDIRECT_URI=http://127.0.0.1:8989/login
// y librespot oauth_sync.rs - solo 127.0.0.1 whitelisteado para el client oficial.
const (
	DefaultClientID = "65b708073fc0480ea92a077233ca87bd" // oficial, NO editable, solo loopback whitelisteado
	OfficialRedirectURI = "http://127.0.0.1:8989/login" // Sonora usa 8989, librespot 8898, desktop 4388
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

func OAuthConfig(redirectURL string, clientID string) *oauth2.Config {
	if clientID == "" {
		clientID = DefaultClientID
	}
	return &oauth2.Config{
		ClientID:    clientID,
		Endpoint:    oauth2.Endpoint{AuthURL: AuthURL, TokenURL: TokenURL},
		RedirectURL: redirectURL,
		Scopes:      Scopes,
	}
}

// IsOfficialClient indica si el redirect debe ser loopback whitelisteado
func IsOfficialClient(clientID string) bool {
	return clientID == "" || clientID == DefaultClientID
}

func RandomState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func RandomVerifier() string {
	b := make([]byte, 64)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func AuthURLWithState(cfg *oauth2.Config, state string) string {
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func AuthURLWithPKCE(cfg *oauth2.Config, state, verifier string) string {
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
}
