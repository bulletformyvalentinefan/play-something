package spotify

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

// Manager Sonora-style: guarda token y Session por spotifyID.
// En Linux crea Session real de go-librespot (Mercury/AP), en Windows es stub que usa Web API.
type Manager struct {
	mu       sync.RWMutex
	tokens   map[string]*oauth2.Token
	profiles map[string]Profile
}

type Profile struct {
	ID          string
	DisplayName string
	Email       string
	Image       string
}

func NewManager() *Manager {
	return &Manager{
		tokens:   make(map[string]*oauth2.Token),
		profiles: make(map[string]Profile),
	}
}

func (m *Manager) Save(userID string, tok *oauth2.Token, p Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[userID] = tok
	m.profiles[userID] = p
}

func (m *Manager) GetToken(userID string) (*oauth2.Token, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tokens[userID]
	return t, ok
}

func (m *Manager) GetProfile(userID string) (Profile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.profiles[userID]
	return p, ok
}

func (m *Manager) Delete(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, userID)
	delete(m.profiles, userID)
}

func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = make(map[string]*oauth2.Token)
	m.profiles = make(map[string]Profile)
}

func (m *Manager) First() (string, *oauth2.Token, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, t := range m.tokens {
		return id, t, true
	}
	return "", nil, false
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.tokens))
	for id := range m.tokens {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) FindUserByToken(token string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, t := range m.tokens {
		if t.AccessToken == token {
			return id, true
		}
	}
	return "", false
}

// SessionSearch is implemented per OS (linux vs stub)
type SearchResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	AlbumCover string `json:"albumCover"`
	DurationMs int    `json:"duration_ms"`
	URI        string `json:"uri"`
}

// Search tries spclient (Session) first, falls back to Web API.
func (m *Manager) Search(ctx context.Context, userID, query string) ([]SearchResult, error) {
	if res, err := m.searchViaSpclient(ctx, userID, query); err == nil && len(res) > 0 {
		return res, nil
	}
	return m.searchViaWebAPI(ctx, userID, query)
}

func (m *Manager) searchViaWebAPI(ctx context.Context, userID, query string) ([]SearchResult, error) {
	tok, ok := m.GetToken(userID)
	if !ok {
		if _, t, ok := m.First(); ok {
			tok = t
		} else {
			return nil, ErrNotLinked
		}
	}
	return webAPISearch(ctx, tok.AccessToken, query)
}

var ErrNotLinked = errNotLinked("not linked")

type errNotLinked string
func (e errNotLinked) Error() string { return string(e) }
