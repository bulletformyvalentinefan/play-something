package handler

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bulletformyvalentinefan/play-something/spotify-proxy/internal/spotify"
)

// TestProxySearch_ConTokenPremium - unit test de integración contra Spotify real.
// Usa tu token premium de localStorage spotify_token.
// Ejecuta: SPOTIFY_TOKEN="BQ..." go test ./internal/handler -run TestProxySearch -v
func TestProxySearch_ConTokenPremium(t *testing.T) {
	token := os.Getenv("SPOTIFY_TOKEN")
	if token == "" {
		t.Skip("SPOTIFY_TOKEN no seteado - copia tu token de DevTools Application > Local Storage > spotify_token tras logearte en http://localhost:5173")
	}
	mgr := spotify.NewManager()
	// No necesitamos Save porque SearchWithToken usa el token directo
	h := NewProxyHandlerWithManager(mgr)

	req := httptest.NewRequest("GET", "/api/v1/spotify/proxy/search?q=pierce%20the%20veil&type=track&limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Search(w, req)

	resp := w.Result()
	if resp.StatusCode == 429 {
		t.Logf("429 rate limit - reintenta en 30s, es normal con Web API, en Linux Docker con spclient no pasa")
		t.Skip("rate limited, prueba luego o usa Docker Linux para spclient")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d, esperado 200", resp.StatusCode)
	}
	// verifica que no sea sin resultados vacío
	body := w.Body.String()
	if len(body) < 50 {
		t.Fatalf("respuesta muy corta: %s", body)
	}
	t.Logf("proxy search OK: %s", body[:200])
}

// TestProxySearch_SinToken_401
func TestProxySearch_SinToken_401(t *testing.T) {
	h := NewProxyHandler()
	req := httptest.NewRequest("GET", "/api/v1/spotify/proxy/search?q=test&type=track", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	if w.Code != 401 {
		t.Fatalf("esperado 401 sin token, got %d", w.Code)
	}
}

func TestProxySearch_CacheHit(t *testing.T) {
	token := os.Getenv("SPOTIFY_TOKEN")
	if token == "" {
		t.Skip("SPOTIFY_TOKEN no seteado")
	}
	mgr := spotify.NewManager()
	h := NewProxyHandlerWithManager(mgr)
	// primera búsqueda (miss)
	req1 := httptest.NewRequest("GET", "/api/v1/spotify/proxy/search?q=muse%20hysteria&type=track&limit=5", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	h.Search(w1, req1)
	if w1.Code == 429 {
		t.Skip("rate limited en primera búsqueda")
	}
	// segunda igual debe venir de cache en Windows dev (X-Cache: HIT) o spclient en Linux
	req2 := httptest.NewRequest("GET", "/api/v1/spotify/proxy/search?q=muse%20hysteria&type=track&limit=5", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	h.Search(w2, req2)
	t.Logf("segunda búsqueda status %d X-Cache=%s X-Source=%s", w2.Code, w2.Header().Get("X-Cache"), w2.Header().Get("X-Source"))
}
