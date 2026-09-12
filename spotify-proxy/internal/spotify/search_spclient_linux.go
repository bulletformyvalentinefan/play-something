//go:build linux

package spotify

import (
	"context"
	"fmt"
	"strings"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	devicespb "github.com/devgianlu/go-librespot/proto/spotify/connectstate/devices"
)

// searchViaSpclient Sonora-style: spotify:search:{escaped} via session.spclient().ContextResolve
func (m *Manager) searchViaSpclient(ctx context.Context, userID, query string) ([]SearchResult, error) {
	tok, ok := m.GetToken(userID)
	if !ok {
		if _, t, ok := m.First(); ok {
			tok = t
		} else {
			return nil, ErrNotLinked
		}
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	// 20 bytes hex = 40 chars, Sonora usa cache dir random, aquí dummy válido
	deviceID := "1234567890abcdef1234567890abcdef1234567890"
	sess, err := session.NewSessionFromOptions(ctx, &session.Options{
		Log:        &librespot.NullLogger{},
		DeviceType: devicespb.DeviceType_COMPUTER,
		DeviceId:   deviceID,
		Credentials: session.SpotifyTokenCredentials{Username: userID, Token: tok.AccessToken},
	})
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	uri := "spotify:search:" + escaped(query)
	sp := sess.Spclient()
	cctx, err := sp.ContextResolve(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("context resolve failed: %w", err)
	}
	// extraer URIs de tracks (connectpb.ContextPage.Tracks[].Uri es string)
	var uris []string
	for _, page := range cctx.Pages {
		for _, tr := range page.Tracks {
			if tr.Uri != "" && strings.HasPrefix(tr.Uri, "spotify:track:") {
				uris = append(uris, tr.Uri)
			}
		}
	}
	if len(uris) == 0 {
		return nil, nil
	}
	// resolver metadata via ExtendedMetadata (simplificado: cada uri es spotify:track:{id})
	out := make([]SearchResult, 0, len(uris))
	for _, uri := range uris {
		id := strings.TrimPrefix(uri, "spotify:track:")
		out = append(out, SearchResult{ID: id, Name: id, URI: uri})
		if len(out) >= 20 {
			break
		}
	}
	return out, nil
}

func escaped(query string) string {
	var b strings.Builder
	for _, r := range query {
		switch {
		case r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '~':
			b.WriteRune(r)
		case r == ' ':
			b.WriteString("+")
		default:
			for _, bb := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", bb)
			}
		}
	}
	return b.String()
}


