//go:build !linux

package spotify

import "context"

func (m *Manager) searchViaSpclient(ctx context.Context, userID, query string) ([]SearchResult, error) {
	return nil, ErrNotLinked
}

func (m *Manager) searchViaSpclientWithToken(ctx context.Context, token, query string) ([]SearchResult, error) {
	return nil, ErrNotLinked
}
