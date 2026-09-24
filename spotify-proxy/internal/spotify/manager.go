package spotify

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/devgianlu/go-librespot/spclient"
	"golang.org/x/oauth2"
)

type Manager struct {
	mu       sync.RWMutex
	tokens   map[string]*oauth2.Token
	profiles map[string]Profile
	sessions map[string]*spclientSession
}

type spclientSession struct {
	sp    *spclient.Spclient
	token string
	clean func()
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
		sessions: make(map[string]*spclientSession),
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
	if s, ok := m.sessions[userID]; ok {
		s.clean()
		delete(m.sessions, userID)
	}
}

func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		s.clean()
	}
	m.tokens = make(map[string]*oauth2.Token)
	m.profiles = make(map[string]Profile)
	m.sessions = make(map[string]*spclientSession)
}

func (m *Manager) First() (string, *oauth2.Token, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, t := range m.tokens {
		return id, t, true
	}
	return "", nil, false
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

// Warmup crea la sesión spclient en background para que la primera búsqueda
// no pague el costo de conexión (~3s). Se llama tras el login OAuth.
func (m *Manager) Warmup(userID string) {
	tok, ok := m.GetToken(userID)
	if !ok {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		owner := userID
		if id, found := m.FindUserByToken(tok.AccessToken); found {
			owner = id
		}
		_, _, _ = m.getOrCreateSpclient(ctx, owner, tok.AccessToken)
	}()
}

func (m *Manager) getOrCreateSpclient(ctx context.Context, userID, accessToken string) (*spclient.Spclient, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[userID]; ok && s.token == accessToken {
		return s.sp, func() {}, nil
	}

	if s, ok := m.sessions[userID]; ok {
		s.clean()
		delete(m.sessions, userID)
	}

	profile := m.profiles[userID]
	username := profile.ID
	if username == "" {
		username = userID
	}

	sp, cleanup, err := NewSpclientSession(ctx, username, accessToken)
	if err != nil {
		return nil, nil, err
	}

	m.sessions[userID] = &spclientSession{sp: sp, token: accessToken, clean: cleanup}
	log.Printf("[manager] spclient session created for %s", userID)
	return sp, func() {}, nil
}

type SearchResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	AlbumCover string `json:"albumCover"`
	DurationMs int    `json:"duration_ms"`
	URI        string `json:"uri"`
}

func (m *Manager) Search(ctx context.Context, userID, query string, limit int) ([]SearchResult, error) {
	tok, ok := m.GetToken(userID)
	if !ok {
		var t *oauth2.Token
		if userID, t, ok = m.First(); !ok {
			return nil, ErrNotLinked
		}
		tok = t
	}
	// Key the cached session by the token owner, not the requested userID,
	// so a fallback token never gets stored under the wrong user.
	if owner, found := m.FindUserByToken(tok.AccessToken); found {
		userID = owner
	}
	sp, _, err := m.getOrCreateSpclient(ctx, userID, tok.AccessToken)
	if err != nil {
		return nil, err
	}
	return searchSpclient(ctx, sp, query, limit)
}

func (m *Manager) SearchWithToken(ctx context.Context, token, query string, limit int) ([]SearchResult, error) {
	username, _ := m.FindUserByToken(token)
	if username == "" {
		username = "spotify-user"
	}
	sp, _, err := m.getOrCreateSpclient(ctx, username, token)
	if err != nil {
		return nil, err
	}
	return searchSpclient(ctx, sp, query, limit)
}

var ErrNotLinked = errNotLinked("not linked")

type errNotLinked string

func (e errNotLinked) Error() string { return string(e) }
