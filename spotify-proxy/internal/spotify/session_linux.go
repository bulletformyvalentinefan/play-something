//go:build linux

package spotify

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	"github.com/devgianlu/go-librespot/spclient"
	"github.com/sirupsen/logrus"
)

// linuxService extiende Service con sesiones reales en Linux.
type linuxService struct {
	log      *logrus.Logger
	mu       sync.Mutex
	sessions map[string]*userSession
}

type userSession struct {
	sess     *session.Session
	sp       *spclient.Spclient
	deviceID string
}

func newLinuxService(log *logrus.Logger) *linuxService {
	return &linuxService{log: log, sessions: make(map[string]*userSession)}
}

func (s *linuxService) EnsureSession(ctx context.Context, userID string, opts session.Options) (*userSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if us, ok := s.sessions[userID]; ok && us.sess != nil {
		return us, nil
	}
	sess, err := session.NewSessionFromOptions(session.Options{
		DeviceType:  opts.DeviceType,
		DeviceId:    opts.DeviceId,
		DeviceName:  opts.DeviceName,
		Bitrate:     opts.Bitrate,
		Credentials: opts.Credentials,
		Log:         s.log,
	})
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	httpClient := &http.Client{}
	accessTokenFn := sess.Login5().AccessToken()
	sp, err := spclient.NewSpclient(ctx, s.log, httpClient, func(ctx context.Context) string { return "spclient.wg.spotify.com" }, accessTokenFn, opts.DeviceId, sess.ClientToken())
	if err != nil {
		return nil, fmt.Errorf("new spclient: %w", err)
	}
	us := &userSession{sess: sess, sp: sp, deviceID: opts.DeviceId}
	s.sessions[userID] = us
	return us, nil
}

func (s *linuxService) GetLogin5Token(ctx context.Context, userID string, force bool) (string, error) {
	s.mu.Lock()
	us, ok := s.sessions[userID]
	s.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("no session for user %s", userID)
	}
	return us.sess.Login5().AccessToken()(ctx, force)
}

func (s *linuxService) ProxyRequest(ctx context.Context, userID, method, path string, header http.Header, body []byte) (*http.Response, error) {
	s.mu.Lock()
	us, ok := s.sessions[userID]
	s.mu.Unlock()
	if !ok || us.sp == nil {
		return nil, fmt.Errorf("no session for user %s", userID)
	}
	return us.sp.Request(ctx, method, path, nil, header, body)
}

var _ librespot.Logger = (*logrus.Logger)(nil)
var _ = newLinuxService
var _ = (*linuxService)(nil)
