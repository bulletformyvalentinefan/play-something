package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func webAPISearch(ctx context.Context, token, query string) ([]SearchResult, error) {
	v := url.Values{}
	v.Set("q", query)
	v.Set("type", "track")
	v.Set("limit", "20")
	u := "https://api.spotify.com/v1/search?" + v.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		ra := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("Spotify rate limit — reintenta en %ss (spclient evita esto)", ra)
	}
	if resp.StatusCode != 200 {
		var e struct{ Error struct{ Message string `json:"message"` } `json:"error"`}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error.Message != "" {
			return nil, fmt.Errorf("%s", e.Error.Message)
		}
		return nil, fmt.Errorf("search failed %d", resp.StatusCode)
	}
	var data struct {
		Tracks struct {
			Items []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				URI      string `json:"uri"`
				Duration int    `json:"duration_ms"`
				Artists []struct{ Name string `json:"name"` } `json:"artists"`
				Album struct {
					Images []struct{ URL string `json:"url"` } `json:"images"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(data.Tracks.Items))
	for _, t := range data.Tracks.Items {
		artist := ""
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}
		cover := ""
		if len(t.Album.Images) > 0 {
			cover = t.Album.Images[0].URL
		}
		out = append(out, SearchResult{ID: t.ID, Name: t.Name, Artist: artist, AlbumCover: cover, DurationMs: t.Duration, URI: t.URI})
	}
	return out, nil
}
