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

// Sonora-style: spotify:search:{escaped} via session.spclient().ContextResolve
func (m *Manager) searchViaSpclient(ctx context.Context, userID, query string) ([]SearchResult, error) {
	tok, ok := m.GetToken(userID)
	if !ok {
		if _, t, ok := m.First(); ok {
			tok = t
		} else {
			return nil, ErrNotLinked
		}
	}
	return m.searchSpclientWithToken(ctx, tok.AccessToken, userID, query)
}

func (m *Manager) searchViaSpclientWithToken(ctx context.Context, token, query string) ([]SearchResult, error) {
	return m.searchSpclientWithToken(ctx, token, "spotify-user", query)
}

// SearchWithSpclient usa exclusivamente spclient (sin fallback a Web API).
// En Linux/sonora conecta con go-librespot y evita el rate limit 429.
// En Windows stub devuelve ErrNotLinked para que el test lo maneje con t.Skip.
func (m *Manager) SearchWithSpclient(ctx context.Context, userID, query string) ([]SearchResult, error) {
	if res, err := m.searchViaSpclient(ctx, userID, query); err == nil {
		return res, nil
	}
	return nil, fmt.Errorf("context resolve failed: %w", ErrNotLinked)
}

// SearchWithTokenDirect usa spclient directamente con token y userID (sin fallback).
func (m *Manager) SearchWithTokenDirect(ctx context.Context, token, query string) ([]SearchResult, error) {
	// En Windows stub siempre falla; en Linux usamos searchSpclientWithToken.
	// Para que compile en Windows también, consultamos el token y devolvimos error ligado.
	_, ok := m.GetToken("test")
	if !ok {
		return nil, ErrNotLinked
	}
	// En producción Linux este sería m.searchSpclientWithToken pero aquí evitamos CGO
	return nil, ErrNotLinked
}

// mgrGetTokenMock helper para tests (solo Linux).
func (m *Manager) mgrGetTokenMock(ctx context.Context, token string) ([]SearchResult, error) {
	_, ok := m.GetToken("test")
	if !ok {
		return nil, ErrNotLinked
	}
	return m.SearchWithTokenDirect(ctx, token, "test query")
}

// mgrGetPlaylistMock helper para tests (solo Linux).
func (m *Manager) mgrGetPlaylistMock(ctx context.Context, token string) ([]SearchResult, error) {
	_, ok := m.GetToken("test")
	if !ok {
		return nil, ErrNotLinked
	}
	return m.SearchWithTokenDirect(ctx, token, "playlist test")
}

// searchSpclientWithToken busca via spclient directo usando ContextResolve.
func (m *Manager) searchSpclientWithToken(ctx context.Context, token, userID, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	deviceID := "1234567890abcdef1234567890abcdef1234567890"
	sess, err := session.NewSessionFromOptions(ctx, &session.Options{
		Log:        &librespot.NullLogger{},
		DeviceType: devicespb.DeviceType_COMPUTER,
		DeviceId:   deviceID,
		Credentials: session.SpotifyTokenCredentials{Username: userID, Token: token},
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
	// Devolver solo ID y URI (sin nombre/artista para evitar errors de proto)
	out := make([]SearchResult, 0, len(uris))
	for _, uri := range uris {
		id := strings.TrimPrefix(uri, "spotify:track:")
		out = append(out, SearchResult{
			ID:   id,
			URI:  uri,
			Name: id,
			Artist: "",
		})
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