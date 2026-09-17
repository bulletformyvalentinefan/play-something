package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"crypto/rand"
	"encoding/base64"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSpclientLogin(t *testing.T) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	if clientID == "" {
		clientID = "65b708073fc0480ea92a077233ca87bd"
	}
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if redirectURI == "" {
		if clientID == "65b708073fc0480ea92a077233ca87bd" {
			redirectURI = "http://127.0.0.1:8989/login"
		} else {
			redirectURI = "http://127.0.0.1:18989/callback"
		}
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		t.Fatalf("invalid redirect URI: %v", err)
	}

	cfg := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
		RedirectURL: redirectURI,
		Scopes: []string{
			"user-read-private",
			"user-read-email",
			"playlist-read-private",
			"playlist-read-collaborative",
			"user-library-read",
			"streaming",
		},
	}

	state := randomB64(32)
	verifier := randomB64(64)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(parsed.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("error") != "" {
			errCh <- fmt.Errorf("spotify: %s", r.URL.Query().Get("error"))
			http.Error(w, "error", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "no code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h1>Login OK! Cerrar pestana.</h1></body></html>")
	})

	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		t.Fatalf("cannot listen on %s: %v", parsed.Host, err)
	}
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	time.Sleep(150 * time.Millisecond)

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	t.Logf("Opening browser for Spotify login...")
	t.Logf("URL: %s", authURL)

	if err := openBrowser(authURL); err != nil {
		t.Logf("Could not open browser: %v", err)
		t.Logf("Open manually: %s", authURL)
	}

	t.Logf("Waiting for login callback on %s ...", redirectURI)

	var code string
	select {
	case code = <-codeCh:
		t.Logf("Got authorization code")
	case err := <-errCh:
		t.Fatalf("callback error: %v", err)
	case <-time.After(120 * time.Second):
		t.Fatal("timed out (120s)")
	}

	token, err := cfg.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("token exchange failed: %v", err)
	}
	t.Logf("access_token obtained (expires: %s)", token.Expiry.Format(time.RFC3339))

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "https://api.spotify.com/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("profile fetch failed: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Profile response %d: %s", resp.StatusCode, string(bodyBytes))
	var profile struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Product     string `json:"product"`
	}
	_ = json.Unmarshal(bodyBytes, &profile)
	t.Logf("User: %s (%s) — Product: %s", profile.ID, profile.DisplayName, profile.Product)
	if profile.Product != "premium" {
		t.Logf("WARNING: not Premium (%s) — playback may not work", profile.Product)
	}

	t.Logf("Creating librespot session + searching via spclient...")
	results, err := SearchViaSpclient(context.Background(), token.AccessToken, profile.ID, "pierce the veil")
	if err != nil {
		t.Fatalf("spclient search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results from spclient")
	}
	t.Logf("SUCCESS: %d results:", len(results))
	for i, r := range results {
		if i >= 10 {
			t.Logf("  ... and %d more", len(results)-10)
			break
		}
		t.Logf("  %d. %s — %s [%s] (%dms) %s", i+1, r.Name, r.Artist, r.Album, r.DurationMs, r.URI)
	}
}

func randomB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", strings.ReplaceAll(url, "&", "^&")).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
