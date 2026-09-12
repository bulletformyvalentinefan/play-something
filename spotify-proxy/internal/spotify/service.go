package spotify

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Service expone operaciones Spotify vía go-librespot (ingeniería inversa).
// Implementación real solo en linux (CGO). En Windows/mac stub para vet.
type Service struct {
	log *logrus.Logger
}

func NewService(log *logrus.Logger) *Service {
	return &Service{log: log}
}

// GetLogin5Token devuelve access_token via login5 (force renueva).
func (s *Service) GetLogin5Token(_ context.Context, userID string, _ bool) (string, error) {
	return "", notImplemented(userID)
}

// ProxyRequest hace forward autenticado a api.spotify.com o spclient.wg.spotify.com
func (s *Service) ProxyRequest(_ context.Context, userID, method, path string, header http.Header, body []byte) (*http.Response, error) {
	return nil, notImplemented(userID)
}

func notImplemented(userID string) error {
	return &NotImplementedError{UserID: userID}
}

type NotImplementedError struct{ UserID string }

func (e *NotImplementedError) Error() string {
	return "spotify service not implemented on this OS - build on linux with CGO (see Dockerfile)"
}
