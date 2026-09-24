package spotify

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"

	playlist4pb "github.com/devgianlu/go-librespot/proto/spotify/playlist4"
	"github.com/devgianlu/go-librespot/spclient"
	"google.golang.org/protobuf/proto"
)

type PlaylistSummary struct {
	ID          string
	URI         string
	Name        string
	Description string
	Owner       string
	Public      bool
	CoverURL    string
	Total       int
}

type PlaylistFull struct {
	PlaylistSummary
	Tracks []SearchResult
}

func playlistIDFromURI(uri string) string {
	return strings.TrimPrefix(uri, "spotify:playlist:")
}

func playlistCoverURL(pic []byte) string {
	if len(pic) == 0 {
		return ""
	}
	return "https://i.scdn.co/image/" + hex.EncodeToString(pic)
}

// fetchSelectedContent trae una playlist (o el rootlist) vía spclient.
func fetchSelectedContent(ctx context.Context, sp *spclient.Spclient, path string) (*playlist4pb.SelectedListContent, error) {
	resp, err := sp.Request(ctx, "GET", path, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("playlist %d: %s", resp.StatusCode, string(body))
	}
	var content playlist4pb.SelectedListContent
	if err := proto.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf("decode playlist: %w", err)
	}
	return &content, nil
}

func summaryFromContent(uri string, content *playlist4pb.SelectedListContent, public bool) PlaylistSummary {
	attrs := content.GetAttributes()
	return PlaylistSummary{
		ID:          playlistIDFromURI(uri),
		URI:         uri,
		Name:        attrs.GetName(),
		Description: attrs.GetDescription(),
		Owner:       content.GetOwnerUsername(),
		Public:      public,
		CoverURL:    playlistCoverURL(attrs.GetPicture()),
		Total:       int(content.GetLength()),
	}
}

// UserPlaylists lista las playlists del usuario vía rootlist spclient.
// Sin Web API: funciona aunque api.spotify.com rate-limitee el token.
func (m *Manager) UserPlaylists(ctx context.Context, token string, limit, offset int) ([]PlaylistSummary, int, error) {
	username, _ := m.FindUserByToken(token)
	if username == "" {
		username = "spotify-user"
	}
	sess, err := m.getOrCreateSpclient(ctx, username, token)
	if err != nil {
		return nil, 0, err
	}
	// El username real sale del AP (vale aunque /v1/me dé 429).
	if sess.Username != "" {
		username = sess.Username
	}

	root, err := fetchSelectedContent(ctx, sess.Sp, "/playlist/v2/user/"+username+"/rootlist")
	if err != nil {
		return nil, 0, err
	}
	items := root.GetContents().GetItems()
	total := len(items)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := items[offset:end]

	out := make([]PlaylistSummary, len(page))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, 6)
	for i, it := range page {
		wg.Add(1)
		go func(i int, it *playlist4pb.Item) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			uri := it.GetUri()
			content, err := fetchSelectedContent(ctx, sess.Sp, "/playlist/v2/playlist/"+playlistIDFromURI(uri))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				out[i] = PlaylistSummary{ID: playlistIDFromURI(uri), URI: uri, Name: playlistIDFromURI(uri)}
				return
			}
			out[i] = summaryFromContent(uri, content, it.GetAttributes().GetPublic())
		}(i, it)
	}
	wg.Wait()
	if firstErr != nil && len(page) > 0 {
		empty := true
		for _, s := range out {
			if s.Name != "" && s.Name != s.ID {
				empty = false
				break
			}
		}
		if empty {
			return nil, 0, firstErr
		}
	}
	return out, total, nil
}

// PlaylistDetail trae una playlist con sus tracks enriquecidos vía spclient.
func (m *Manager) PlaylistDetail(ctx context.Context, token, playlistID string, limit, offset int) (*PlaylistFull, error) {
	username, _ := m.FindUserByToken(token)
	if username == "" {
		username = "spotify-user"
	}
	sess, err := m.getOrCreateSpclient(ctx, username, token)
	if err != nil {
		return nil, err
	}

	content, err := fetchSelectedContent(ctx, sess.Sp, "/playlist/v2/playlist/"+playlistID)
	if err != nil {
		return nil, err
	}
	full := &PlaylistFull{
		PlaylistSummary: summaryFromContent("spotify:playlist:"+playlistID, content, false),
	}

	var uris []string
	for _, it := range content.GetContents().GetItems() {
		if strings.HasPrefix(it.GetUri(), "spotify:track:") {
			uris = append(uris, it.GetUri())
		}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(uris) {
		offset = len(uris)
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	end := offset + limit
	if end > len(uris) {
		end = len(uris)
	}
	full.Tracks = buildTrackResults(uris[offset:end], enrichTrackMetadata(ctx, sess.Sp, uris[offset:end]))
	return full, nil
}
