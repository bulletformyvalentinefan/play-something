//go:build !linux

package spotify

import "context"

func (m *Manager) searchViaSpclient(ctx context.Context, userID, query string) ([]SearchResult, error) {
	return nil, ErrNotLinked
}
