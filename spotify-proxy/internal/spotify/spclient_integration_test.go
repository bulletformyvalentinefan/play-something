package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
	"google.golang.org/protobuf/proto"

	librespot "github.com/devgianlu/go-librespot"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	playlist4pb "github.com/devgianlu/go-librespot/proto/spotify/playlist4"
)

func loginPremiumToken(t *testing.T) (*oauth2.Token, string) {
	t.Helper()
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

	username := profile.ID
	if username == "" {
		username = "spotify-user"
		t.Logf("WARNING: profile.ID empty (Web API rate limit?) — using fallback username %q", username)
	}
	return token, username
}

func TestSpclientLogin(t *testing.T) {
	token, username := loginPremiumToken(t)

	t.Logf("--- RAW context-resolve dump ---")
	rawSess, err := NewSpclientSession(context.Background(), username, token.AccessToken)
	if err != nil {
		t.Fatalf("raw spclient session failed: %v", err)
	}
	defer rawSess.Close()
	rawSp := rawSess.Sp

	rawResp, err := rawSp.Request(context.Background(), "GET", "/context-resolve/v1/spotify:search:pierce+the+veil", nil, nil, nil)
	if err != nil {
		t.Fatalf("raw context resolve failed: %v", err)
	}
	rawBody, _ := io.ReadAll(rawResp.Body)
	rawResp.Body.Close()
	t.Logf("raw context-resolve: status=%d bytes=%d", rawResp.StatusCode, len(rawBody))
	if rawResp.StatusCode != 200 {
		t.Fatalf("raw context resolve status %d: %s", rawResp.StatusCode, string(rawBody))
	}
	var rawCtx connectpb.Context
	if err := json.Unmarshal(rawBody, &rawCtx); err != nil {
		t.Fatalf("raw context decode failed: %v", err)
	}
	t.Logf("raw context: pages=%d", len(rawCtx.Pages))
	var rawURIs []string
	for pi, page := range rawCtx.Pages {
		t.Logf("raw page %d: tracks=%d", pi, len(page.Tracks))
		for ti, tr := range page.Tracks {
			if ti < 3 {
				t.Logf("raw track %d: uri=%q metadata=%v", ti, tr.Uri, tr.Metadata)
			}
			if strings.HasPrefix(tr.Uri, "spotify:track:") && len(rawURIs) < 3 {
				rawURIs = append(rawURIs, tr.Uri)
			}
		}
	}

	t.Logf("--- RAW extended-metadata dump (%d uris) ---", len(rawURIs))
	rawReqs := make([]*extmetadatapb.EntityRequest, len(rawURIs))
	for i, uri := range rawURIs {
		rawReqs[i] = &extmetadatapb.EntityRequest{
			EntityUri: uri,
			Query: []*extmetadatapb.ExtensionQuery{{
				ExtensionKind: extmetadatapb.ExtensionKind_TRACK_V4,
			}},
		}
	}
	rawMeta, err := rawSp.ExtendedMetadata(context.Background(), &extmetadatapb.BatchedEntityRequest{
		EntityRequest: rawReqs,
	})
	if err != nil {
		t.Fatalf("raw extended metadata failed: %v", err)
	}
	t.Logf("raw extended-metadata: arrays=%d", len(rawMeta.GetExtendedMetadata()))
	for _, arr := range rawMeta.GetExtendedMetadata() {
		t.Logf("raw array: kind=%v items=%d", arr.GetExtensionKind(), len(arr.GetExtensionData()))
		for _, ed := range arr.GetExtensionData() {
			t.Logf("raw entity: uri=%q status=%d type=%q",
				ed.GetEntityUri(), ed.GetHeader().GetStatusCode(), ed.GetExtensionData().GetTypeUrl())
			var rtrack metadatapb.Track
			if err := ed.GetExtensionData().UnmarshalTo(&rtrack); err != nil {
				t.Fatalf("raw track unmarshal failed: %v", err)
			}
			var anames []string
			for _, a := range rtrack.GetArtist() {
				anames = append(anames, a.GetName())
			}
			coverN, coverGroupN := 0, 0
			if alb := rtrack.GetAlbum(); alb != nil {
				coverN = len(alb.GetCover())
				if cg := alb.GetCoverGroup(); cg != nil {
					coverGroupN = len(cg.GetImage())
				}
			}
			t.Logf("raw track: name=%q artists=%q album=%q duration=%d popularity=%d explicit=%v covers=%d coverGroup=%d files=%d",
				rtrack.GetName(), strings.Join(anames, ", "), rtrack.GetAlbum().GetName(),
				rtrack.GetDuration(), rtrack.GetPopularity(), rtrack.GetExplicit(),
				coverN, coverGroupN, len(rtrack.GetFile()))
		}
	}

	t.Logf("Creating librespot session + searching via spclient...")
	results, err := SearchViaSpclient(context.Background(), token.AccessToken, username, "pierce the veil")
	if err != nil {
		t.Fatalf("spclient search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results from spclient")
	}
	if len(results) > 20 {
		t.Errorf("expected at most 20 results, got %d", len(results))
	}

	var withName, withArtist, withAlbum, withDuration, withCover int
	seen := make(map[string]bool)
	for i, r := range results {
		if r.URI == "" || !strings.HasPrefix(r.URI, "spotify:track:") {
			t.Errorf("result %d: bad URI %q", i, r.URI)
			continue
		}
		if want := strings.TrimPrefix(r.URI, "spotify:track:"); r.ID != want {
			t.Errorf("result %d: ID %q does not match URI %q", i, r.ID, r.URI)
		}
		if seen[r.URI] {
			t.Errorf("result %d: duplicate URI %q", i, r.URI)
		}
		seen[r.URI] = true
		if r.Name != "" && r.Name != r.ID {
			withName++
		}
		if r.Artist != "" {
			withArtist++
		}
		if r.Album != "" {
			withAlbum++
		}
		if r.DurationMs > 0 {
			withDuration++
		}
		if strings.HasPrefix(r.AlbumCover, "https://i.scdn.co/image/") {
			withCover++
		}
		if i < 10 {
			t.Logf("  %d. %s — %s [%s] (%dms) %s", i+1, r.Name, r.Artist, r.Album, r.DurationMs, r.URI)
		}
	}
	if len(results) > 10 {
		t.Logf("  ... and %d more", len(results)-10)
	}
	t.Logf("enriched: name=%d/%d artist=%d/%d album=%d/%d duration=%d/%d cover=%d/%d",
		withName, len(results), withArtist, len(results), withAlbum, len(results),
		withDuration, len(results), withCover, len(results))

	n := len(results)
	if withName*100/n < 80 {
		t.Errorf("expected >=80%% results with enriched name, got %d/%d", withName, n)
	}
	if withArtist*100/n < 80 {
		t.Errorf("expected >=80%% results with artist, got %d/%d", withArtist, n)
	}
	if withDuration != n {
		t.Errorf("expected all results with duration > 0, got %d/%d", withDuration, n)
	}

	t.Logf("--- stream smoke (audio real vía token) ---")
	streamURI := results[0].URI
	sreq := httptest.NewRequest("GET", "/stream?uri="+url.QueryEscape(streamURI), nil)
	sreq.Header.Set("Range", "bytes=0-4095")
	srec := httptest.NewRecorder()
	if err := streamTrack(context.Background(), rawSess, streamURI, srec, sreq); err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	res := srec.Result()
	ct := res.Header.Get("Content-Type")
	t.Logf("stream: status=%d content-type=%s accept-ranges=%s", res.StatusCode, ct, res.Header.Get("Accept-Ranges"))
	if ct != "audio/ogg" && ct != "audio/mpeg" {
		t.Errorf("unexpected stream content-type %q", ct)
	}
	if res.StatusCode != http.StatusPartialContent && res.StatusCode != http.StatusOK {
		t.Errorf("unexpected stream status %d", res.StatusCode)
	}
	streamBody, _ := io.ReadAll(res.Body)
	t.Logf("stream: first bytes=%d", len(streamBody))
	if len(streamBody) == 0 {
		t.Errorf("empty stream body")
	} else if ct == "audio/ogg" && (len(streamBody) < 4 || string(streamBody[0:4]) != "OggS") {
		t.Errorf("stream ogg sin header OggS (página de metadata no salteada)")
	}

	t.Logf("--- playlists vía spclient ---")
	mgr := NewManager()
	mgr.Save("probe-user", token, Profile{})
	lists, total, err := mgr.UserPlaylists(context.Background(), token.AccessToken, 5, 0)
	if err != nil {
		t.Fatalf("user playlists failed: %v", err)
	}
	t.Logf("playlists: total=%d", total)
	if total == 0 || len(lists) == 0 {
		t.Fatalf("sin playlists en rootlist")
	}
	for i, l := range lists {
		t.Logf("  %d. %s [%s] tracks=%d cover=%s", i+1, l.Name, l.ID, l.Total, l.CoverURL)
		if l.Name == "" || l.Name == l.ID {
			t.Errorf("playlist %d sin nombre enriquecido", i)
		}
	}

	detail, err := mgr.PlaylistDetail(context.Background(), token.AccessToken, lists[0].ID, 5, 0)
	if err != nil {
		t.Fatalf("playlist detail failed: %v", err)
	}
	t.Logf("detail: %q tracks=%d/%d", detail.Name, len(detail.Tracks), detail.Total)
	if detail.Name == "" {
		t.Errorf("playlist detail sin nombre")
	}
	if len(detail.Tracks) == 0 {
		t.Errorf("playlist detail sin tracks")
	}
	for i, tr := range detail.Tracks {
		if i >= 3 {
			break
		}
		t.Logf("  %d. %s — %s", i+1, tr.Name, tr.Artist)
		if tr.Name == "" || tr.Name == tr.ID {
			t.Errorf("track %d sin enriquecer", i)
		}
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

// TestPlaylistProbe sondea endpoints de playlists vía spclient con el token
// Premium. Solo diagnostica (logs); no falla salvo error de transporte.
func TestPlaylistProbe(t *testing.T) {
	token, username := loginPremiumToken(t)

	sess, err := NewSpclientSession(context.Background(), username, token.AccessToken)
	if err != nil {
		t.Fatalf("session failed: %v", err)
	}
	defer sess.Close()
	sp := sess.Sp

	probe := func(method, path string, body []byte) {
		resp, err := sp.Request(context.Background(), method, path, nil, nil, body)
		if err != nil {
			t.Logf("PROBE %s %s: err=%v", method, path, err)
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		snip := string(b)
		if len(snip) > 300 {
			snip = snip[:300]
		}
		t.Logf("PROBE %s %s: status=%d ct=%q len=%d body=%q", method, path, resp.StatusCode, resp.Header.Get("Content-Type"), len(b), snip)
	}

	t.Logf("PROBE: username AP=%q profile=%q", sess.Username, username)
	if sess.Username != "" {
		probe("GET", "/playlist/v2/user/"+sess.Username+"/rootlist", nil)
	} else {
		t.Logf("PROBE: sin username real, salto rootlist")
	}

	if sess.Username != "" {
		resp, err := sess.Sp.Request(context.Background(), "GET", "/playlist/v2/user/"+sess.Username+"/rootlist", nil, nil, nil)
		if err != nil {
			t.Logf("PROBE rootlist parse: err=%v", err)
		} else {
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			var root playlist4pb.SelectedListContent
			if err := proto.Unmarshal(b, &root); err != nil {
				t.Logf("PROBE rootlist parse: unmarshal err=%v", err)
			} else {
				t.Logf("PROBE rootlist parse: length=%d items=%d", root.GetLength(), len(root.GetContents().GetItems()))
				var plURIs []string
				for i, it := range root.GetContents().GetItems() {
					if i < 5 {
						t.Logf("PROBE rootlist item %d: uri=%q", i, it.GetUri())
					}
					if strings.HasPrefix(it.GetUri(), "spotify:playlist:") && len(plURIs) < 3 {
						plURIs = append(plURIs, it.GetUri())
					}
				}
				if len(plURIs) > 0 {
					reqs := make([]*extmetadatapb.EntityRequest, len(plURIs))
					for i, uri := range plURIs {
						reqs[i] = &extmetadatapb.EntityRequest{
							EntityUri: uri,
							Query:     []*extmetadatapb.ExtensionQuery{{ExtensionKind: extmetadatapb.ExtensionKind_PLAYLIST_ATTRIBUTES_V2}},
						}
					}
					meta, err := sess.Sp.ExtendedMetadata(context.Background(), &extmetadatapb.BatchedEntityRequest{EntityRequest: reqs})
					if err != nil {
						t.Logf("PROBE playlist attrs: err=%v", err)
					} else {
						for _, arr := range meta.GetExtendedMetadata() {
							for _, ed := range arr.GetExtensionData() {
								t.Logf("PROBE playlist attrs: uri=%q status=%d type=%q",
									ed.GetEntityUri(), ed.GetHeader().GetStatusCode(), ed.GetExtensionData().GetTypeUrl())
								var attrs playlist4pb.ListAttributes
								if err := ed.GetExtensionData().UnmarshalTo(&attrs); err != nil {
									t.Logf("PROBE playlist attrs: unmarshal ListAttributes err=%v", err)
									continue
								}
								t.Logf("PROBE playlist attrs: name=%q desc=%q picture=%dB",
									attrs.GetName(), attrs.GetDescription(), len(attrs.GetPicture()))
							}
						}
					}
				}
			}
		}
	}

	plID := "37i9dQZF1DXcBWIGoYBM5M" // Today's Top Hits (pública)
	probe("GET", "/playlist/v2/playlist/"+plID, nil)

	resp, err := sess.Sp.Request(context.Background(), "GET", "/playlist/v2/playlist/"+plID, nil, nil, nil)
	if err != nil {
		t.Logf("PROBE playlist parse: err=%v", err)
	} else {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var content playlist4pb.SelectedListContent
		if err := proto.Unmarshal(b, &content); err != nil {
			t.Logf("PROBE playlist parse: unmarshal err=%v", err)
		} else {
			t.Logf("PROBE playlist parse: name=%q length=%d items=%d owner=%q",
				content.GetAttributes().GetName(), content.GetLength(),
				len(content.GetContents().GetItems()), content.GetOwnerUsername())
			for i, it := range content.GetContents().GetItems() {
				if i >= 5 {
					break
				}
				t.Logf("PROBE playlist item %d: uri=%q", i, it.GetUri())
			}
		}
	}

	sid, err := librespot.SpotifyIdFromUri("spotify:playlist:" + plID)
	if err != nil {
		t.Logf("PROBE signals: bad id: %v", err)
	} else {
		res, err := sp.PlaylistSignals(context.Background(), *sid, &playlist4pb.ListSignals{}, nil)
		if err != nil {
			t.Logf("PROBE signals: err=%v", err)
		} else {
			t.Logf("PROBE signals: length=%d items=%d attrs=%v owner=%q", res.GetLength(), len(res.GetContents().GetItems()), res.GetAttributes(), res.GetOwnerUsername())
			for i, it := range res.GetContents().GetItems() {
				if i >= 5 {
					break
				}
				t.Logf("PROBE signals item %d: uri=%q", i, it.GetUri())
			}
		}
	}
}
