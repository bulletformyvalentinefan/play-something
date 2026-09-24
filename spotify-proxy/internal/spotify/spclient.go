package spotify

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/ap"
	"github.com/devgianlu/go-librespot/apresolve"
	"github.com/devgianlu/go-librespot/login5"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	pbdata "github.com/devgianlu/go-librespot/proto/spotify/clienttoken/data/v0"
	pbhttp "github.com/devgianlu/go-librespot/proto/spotify/clienttoken/http/v0"
	credentialspb "github.com/devgianlu/go-librespot/proto/spotify/login5/v3/credentials"
	"github.com/devgianlu/go-librespot/spclient"
	"google.golang.org/protobuf/proto"
)

// NewSpclientSession creates a go-librespot spclient using the user's Spotify
// Premium access token. This replicates session.NewSessionFromOptions but
// avoids importing the "session" or "player" packages (which pull in CGO
// audio decoders). All packages used here are pure Go.
func NewSpclientSession(ctx context.Context, username, token string) (*spclient.Spclient, func(), error) {
	log := &librespot.NullLogger{}
	client := &http.Client{Timeout: 30 * time.Second}
	deviceId := hex.EncodeToString([]byte("playsomething00000"))

	// 1. Obtain client token
	clientToken, err := retrieveClientToken(ctx, client, deviceId)
	if err != nil {
		return nil, nil, fmt.Errorf("client token: %w", err)
	}

	// 2. Resolve endpoints
	resolver := apresolve.NewApResolver(log, client, false)

	// 3. Connect to access point with user's OAuth token
	apAddr, err := resolver.GetAccesspoint(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve accesspoint: %w", err)
	}
	accesspoint := ap.NewAccesspoint(log, apAddr, deviceId)
	if err := accesspoint.ConnectSpotifyToken(ctx, username, token); err != nil {
		return nil, nil, fmt.Errorf("ap connect: %w", err)
	}

	// 4. Login via login5 (gets internal access token for spclient)
	l5 := login5.NewLogin5(log, client, deviceId, clientToken)
	if err := l5.Login(ctx, &credentialspb.StoredCredential{
		Username: accesspoint.Username(),
		Data:     accesspoint.StoredCredentials(),
	}); err != nil {
		accesspoint.Close()
		return nil, nil, fmt.Errorf("login5: %w", err)
	}

	// 5. Create spclient
	spAddr, err := resolver.GetSpclient(ctx)
	if err != nil {
		accesspoint.Close()
		return nil, nil, fmt.Errorf("resolve spclient: %w", err)
	}
	sp, err := spclient.NewSpclient(ctx, log, client, spAddr, l5.AccessToken(), deviceId, clientToken)
	if err != nil {
		accesspoint.Close()
		return nil, nil, fmt.Errorf("spclient init: %w", err)
	}

	cleanup := func() { accesspoint.Close() }
	return sp, cleanup, nil
}

// SearchViaSpclient creates a fresh spclient session and searches. Used by tests.
// In production, use searchSpclient with a cached session from the Manager.
func SearchViaSpclient(ctx context.Context, token, username, query string) ([]SearchResult, error) {
	sp, cleanup, err := NewSpclientSession(ctx, username, token)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return searchSpclient(ctx, sp, query, DefaultSearchLimit)
}

const (
	// DefaultSearchLimit is used when the caller passes limit <= 0.
	DefaultSearchLimit = 20
	// MaxSearchLimit caps a single search to one ExtendedMetadata batch.
	MaxSearchLimit = 50
)

// searchSpclient searches via spclient ContextResolve + ExtendedMetadata enrichment.
// Expects an already-connected spclient session (avoids reconnect overhead).
func searchSpclient(ctx context.Context, sp *spclient.Spclient, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	uri := "spotify:search:" + escapeQuery(query)
	resp, err := sp.Request(ctx, "GET", fmt.Sprintf("/context-resolve/v1/%s", uri), nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("context resolve request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("context resolve %d: %s", resp.StatusCode, string(respBody))
	}

	var cctx connectpb.Context
	if err := json.NewDecoder(resp.Body).Decode(&cctx); err != nil {
		return nil, fmt.Errorf("decode context: %w", err)
	}

	type trackEntry struct {
		uri string
		idx int
	}
	var entries []trackEntry
	for _, page := range cctx.Pages {
		for _, tr := range page.Tracks {
			if tr.Uri == "" || !strings.HasPrefix(tr.Uri, "spotify:track:") {
				continue
			}
			entries = append(entries, trackEntry{uri: tr.Uri})
			if len(entries) >= limit {
				break
			}
		}
		if len(entries) >= limit {
			break
		}
	}

	if len(entries) == 0 {
		return nil, nil
	}

	uris := make([]string, len(entries))
	for i, e := range entries {
		uris[i] = e.uri
	}

	enriched := enrichTrackMetadata(ctx, sp, uris)

	results := make([]SearchResult, 0, len(entries))
	for _, e := range entries {
		id := strings.TrimPrefix(e.uri, "spotify:track:")
		r := SearchResult{
			ID:  id,
			Name: id,
			URI: e.uri,
		}
		if info, ok := enriched[e.uri]; ok {
			if info.Name != "" {
				r.Name = info.Name
			}
			r.Artist = info.Artist
			r.Album = info.Album
			r.AlbumCover = info.CoverURL
			r.DurationMs = int(info.DurationMs)
		}
		results = append(results, r)
	}
	return results, nil
}

type trackMetadata struct {
	Name       string
	Artist     string
	Album      string
	DurationMs int32
	CoverURL   string
}

func enrichTrackMetadata(ctx context.Context, sp *spclient.Spclient, uris []string) map[string]*trackMetadata {
	if len(uris) == 0 {
		return nil
	}

	entityRequests := make([]*extmetadatapb.EntityRequest, len(uris))
	for i, uri := range uris {
		entityRequests[i] = &extmetadatapb.EntityRequest{
			EntityUri: uri,
			Query: []*extmetadatapb.ExtensionQuery{{
				ExtensionKind: extmetadatapb.ExtensionKind_TRACK_V4,
			}},
		}
	}

	resp, err := sp.ExtendedMetadata(ctx, &extmetadatapb.BatchedEntityRequest{
		EntityRequest: entityRequests,
	})
	if err != nil {
		return nil
	}

	result := make(map[string]*trackMetadata, len(uris))
	for _, arr := range resp.GetExtendedMetadata() {
		for _, ed := range arr.GetExtensionData() {
			if ed.GetExtensionData() == nil {
				continue
			}
			var track metadatapb.Track
			if err := ed.GetExtensionData().UnmarshalTo(&track); err != nil {
				continue
			}

			var anames []string
			for _, a := range track.GetArtist() {
				if n := a.GetName(); n != "" {
					anames = append(anames, n)
				}
			}
			artist := strings.Join(anames, ", ")
			album := ""
			coverURL := ""
		if a := track.GetAlbum(); a != nil {
			album = a.GetName()
			images := a.GetCover()
			if len(images) == 0 {
				if cg := a.GetCoverGroup(); cg != nil {
					images = cg.GetImage()
				}
			}
			for _, img := range images {
				if fid := img.GetFileId(); len(fid) > 0 {
					coverURL = "https://i.scdn.co/image/" + hex.EncodeToString(fid)
					break
				}
			}
		}

			result[ed.GetEntityUri()] = &trackMetadata{
				Name:       track.GetName(),
				Artist:     artist,
				Album:      album,
				DurationMs: track.GetDuration(),
				CoverURL:   coverURL,
			}
		}
	}
	return result
}

func escapeQuery(q string) string {
	var b strings.Builder
	for _, r := range q {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '~':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('+')
		default:
			for _, bb := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", bb)
			}
		}
	}
	return b.String()
}

// retrieveClientToken obtains a client token from Spotify's clienttoken API.
// Replicates session/client_token.go to avoid importing the session package.
func retrieveClientToken(ctx context.Context, client *http.Client, deviceId string) (string, error) {
	body, err := proto.Marshal(&pbhttp.ClientTokenRequest{
		RequestType: pbhttp.ClientTokenRequestType_REQUEST_CLIENT_DATA_REQUEST,
		Request: &pbhttp.ClientTokenRequest_ClientData{
			ClientData: &pbhttp.ClientDataRequest{
				ClientId:      librespot.ClientIdHex,
				ClientVersion: librespot.SpotifyLikeClientVersion(),
				Data: &pbhttp.ClientDataRequest_ConnectivitySdkData{
					ConnectivitySdkData: &pbdata.ConnectivitySdkData{
						DeviceId:             deviceId,
						PlatformSpecificData: librespot.GetPlatformSpecificData(),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://clienttoken.spotify.com/v1/clienttoken", bytes.NewReader(body))
	req.Header.Set("Accept", "application/x-protobuf")
	req.Header.Set("User-Agent", librespot.UserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respPb pbhttp.ClientTokenResponse
	if err := proto.Unmarshal(respBytes, &respPb); err != nil {
		return "", err
	}
	return respPb.GetGrantedToken().GetToken(), nil
}
