package spotify

import (
	"context"
	"os"
	"testing"

	"golang.org/x/oauth2"
)

// TestSearch_WithPremiumToken - unit test real con tu token premium.
// Ejecuta: SPOTIFY_TOKEN="BQ..." go test ./internal/spotify -run TestSearch_WithPremiumToken -v
// Si no hay token, se salta (no rompe CI).
func TestSearch_WithPremiumToken(t *testing.T) {
	token := os.Getenv("SPOTIFY_TOKEN")
	if token == "" {
		t.Skip("SPOTIFY_TOKEN no seteado - logeate con pnpm dev y copia el token de localStorage spotify_token")
	}
	mgr := NewManager()
	mgr.Save("test-user", &oauth2.Token{AccessToken: token}, Profile{ID: "test-user"})
	res, err := mgr.Search(context.Background(), "test-user", "pierce the veil")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("sin resultados para 'pierce the veil' - revisa token premium y scopes")
	}
	t.Logf("search OK: %d tracks, primero: %s - %s (%s)", len(res), res[0].Name, res[0].Artist, res[0].ID)
	for i, r := range res {
		if i >= 3 {
			break
		}
		t.Logf("  [%d] %s - %s (%s)", i, r.Name, r.Artist, r.ID)
	}
}

// TestSearch_SinToken_DebeFallar - verifica que sin token no hay resultados
func TestSearch_SinToken_DebeFallar(t *testing.T) {
	mgr := NewManager()
	_, err := mgr.Search(context.Background(), "no-existe", "test")
	if err == nil {
		t.Fatalf("se esperaba error sin token")
	}
}

func TestWebAPISearch_Direct(t *testing.T) {
	token := os.Getenv("SPOTIFY_TOKEN")
	if token == "" {
		t.Skip("SPOTIFY_TOKEN no seteado")
	}
	res, err := webAPISearch(context.Background(), token, "muse hysteria")
	if err != nil {
		t.Fatalf("webAPISearch failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("webAPISearch sin resultados")
	}
	t.Logf("webAPI OK: %d", len(res))
}
