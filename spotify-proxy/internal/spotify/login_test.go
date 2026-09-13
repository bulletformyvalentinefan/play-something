package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// getTokenByLogin pide login a Spotify y espera a que loguees.
// Si SPOTIFY_TOKEN ya está seteado, lo usa directo. Si no, abre el navegador
// a la URL de auth y hace polling a /auth/status hasta que te loguees.
func getTokenByLogin(t *testing.T) string {
	if tok := os.Getenv("SPOTIFY_TOKEN"); tok != "" {
		return tok
	}
	// intenta obtener token ya logueado en Go (por si hiciste login via pnpm dev)
	if tok := tryGetExistingToken(t); tok != "" {
		t.Logf("usando token existente de Go memory")
		return tok
	}
	t.Log("SPOTIFY_TOKEN no seteado - iniciando login automático...")

	// 1. pedir URL de auth al Go proxy
	resp, err := http.Post("http://127.0.0.1:8081/api/v1/spotify/auth/start", "application/json", nil)
	if err != nil {
		t.Fatalf("no se pudo conectar a Go proxy en :8081 - ¿está corriendo 'go run ./cmd/server'? %v", err)
	}
	defer resp.Body.Close()
	var data struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.URL == "" {
		t.Fatalf("respuesta auth/start inválida")
	}
	t.Logf("Abriendo navegador para login Spotify...")
	t.Logf("URL: %s", data.URL)
	openBrowser(data.URL)

	// 2. polling a /auth/status hasta que linked=true (máx 120s)
	t.Log("Esperando que completes login en el navegador (tienes 120s)...")
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		if tok := tryGetExistingToken(t); tok != "" {
			t.Logf("¡Login detectado! token obtenido")
			return tok
		}
	}
	t.Fatalf("timeout 120s esperando login - ¿completaste el login premium en el navegador?")
	return ""
}

func tryGetExistingToken(t *testing.T) string {
	t.Helper()
	resp, err := http.Get("http://127.0.0.1:8081/api/v1/spotify/auth/token")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	var data struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&data)
	return data.AccessToken
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// TestSearch_AutoLogin - test que pide login automático si no hay token.
func TestSearch_AutoLogin(t *testing.T) {
	token := getTokenByLogin(t)
	mgr := NewManager()
	mgr.Save("test-user", &oauth2.Token{AccessToken: token}, Profile{ID: "test-user"})
	res, err := mgr.Search(context.Background(), "test-user", "pierce the veil")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("sin resultados - token premium válido? query 'pierce the veil' debería devolver tracks")
	}
	t.Logf("✓ search OK: %d tracks", len(res))
	for i, r := range res {
		if i >= 3 {
			break
		}
		t.Logf("  [%d] %s - %s (%s)", i, r.Name, r.Artist, r.ID)
	}
}

func TestProxySearch_AutoLogin(t *testing.T) {
	token := getTokenByLogin(t)
	res, err := webAPISearch(context.Background(), token, "muse hysteria")
	if err != nil {
		t.Fatalf("webAPISearch failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("sin resultados")
	}
	t.Logf("✓ webAPI search OK: %d", len(res))
}
