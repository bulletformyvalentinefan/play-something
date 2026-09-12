package store

import (
	"errors"
	"sync"
	"time"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/crypto"
)

// SpotifyCredentials guarda lo necesario para re-autenticar vía login5.
// accessToken no se persiste en claro largo plazo: se obtiene vía login5 cada vez.
type Credentials struct {
	UserID        string    `json:"user_id"`
	SpotifyUsername string  `json:"spotify_username"`
	DeviceID      string    `json:"device_id"`
	BlobEnc       string    `json:"blob_enc"`        // blob cifrado (si zeroconf)
	AccessTokenEnc string   `json:"access_token_enc"` // último access_token cifrado (cache corto)
	RefreshTokenEnc string  `json:"refresh_token_enc,omitempty"`
	ClientToken   string    `json:"client_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Store interface {
	Save(c Credentials) error
	Get(userID string) (Credentials, error)
	Delete(userID string) error
	List() ([]Credentials, error)
}

var ErrNotFound = errors.New("credentials not found")

// MemoryStore es thread-safe, para dev/test. En prod reemplazar por OracleStore.
type MemoryStore struct {
	mu      sync.RWMutex
	data    map[string]Credentials
	cryptor *crypto.Cryptor
}

func NewMemoryStore(c *crypto.Cryptor) *MemoryStore {
	return &MemoryStore{data: make(map[string]Credentials), cryptor: c}
}

func (m *MemoryStore) Save(c Credentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	m.data[c.UserID] = c
	return nil
}

func (m *MemoryStore) Get(userID string) (Credentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.data[userID]
	if !ok {
		return Credentials{}, ErrNotFound
	}
	return c, nil
}

func (m *MemoryStore) Delete(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, userID)
	return nil
}

func (m *MemoryStore) List() ([]Credentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Credentials, 0, len(m.data))
	for _, v := range m.data {
		out = append(out, v)
	}
	return out, nil
}
