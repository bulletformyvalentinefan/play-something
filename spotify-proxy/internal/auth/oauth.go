package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/oauth2"
)

// Spotify OAuth con el client_id oficial (loopback 127.0.0.1:8989/login
// whitelisteado) o app propia de developer.spotify.com.
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

func AuthURLWithPKCE(cfg *oauth2.Config, state, verifier string) string {
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
}
