package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/linear"
)

const LinearService = "linear"

type Manager struct {
	connections database.Repository[models.ServiceConnection]
	states      database.Repository[models.ServiceOAuthState]
	publicURL   string
	httpClient  *http.Client
}

func NewManager(connections database.Repository[models.ServiceConnection], states database.Repository[models.ServiceOAuthState], publicURL string) *Manager {
	return &Manager{connections: connections, states: states, publicURL: publicURL, httpClient: http.DefaultClient}
}

func (m *Manager) Services() []string { return []string{LinearService} }

func (m *Manager) BeginOAuth(platform string, chatID int, service string) (string, error) {
	if service != LinearService {
		return "", fmt.Errorf("unsupported service %q", service)
	}
	clientID := os.Getenv("LINEAR_CLIENT_ID")
	if clientID == "" {
		return "", errors.New("LINEAR_CLIENT_ID is not configured")
	}
	state := uuid.NewString()
	if err := m.states.Create(&models.ServiceOAuthState{State: state, Platform: platform, ChatID: chatID, Service: service}); err != nil {
		return "", err
	}
	return linear.OAuthConfig{ClientID: clientID, RedirectURL: m.linearRedirectURL()}.AuthCodeURL(state), nil
}

func (m *Manager) linearOAuthConfig() (linear.OAuthConfig, error) {
	clientID, secret := os.Getenv("LINEAR_CLIENT_ID"), os.Getenv("LINEAR_CLIENT_SECRET")
	if clientID == "" || secret == "" {
		return linear.OAuthConfig{}, errors.New("LINEAR_CLIENT_ID/LINEAR_CLIENT_SECRET are not configured")
	}
	return linear.OAuthConfig{ClientID: clientID, ClientSecret: secret, RedirectURL: m.linearRedirectURL(), HTTPClient: m.httpClient}, nil
}

func (m *Manager) linearRedirectURL() string {
	return strings.TrimRight(m.publicURL, "/") + "/oauth/linear/callback"
}

func (m *Manager) CompleteLinearOAuth(ctx context.Context, state, code string) (*models.ServiceConnection, error) {
	states, err := m.states.Where("state = ? AND service = ?", state, LinearService)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, errors.New("invalid or expired oauth state")
	}
	st := states[0]
	oauthConfig, err := m.linearOAuthConfig()
	if err != nil {
		return nil, err
	}
	tok, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	conn := &models.ServiceConnection{Platform: st.Platform, ChatID: st.ChatID, Service: LinearService, AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, ExpiresAt: tok.ExpiresAt}
	existing, _ := m.connections.Where("platform = ? AND chat_id = ? AND service = ?", conn.Platform, conn.ChatID, conn.Service)
	if len(existing) > 0 {
		conn.ID = existing[0].ID
		conn.CreatedAt = existing[0].CreatedAt
		err = m.connections.Update(conn)
	} else {
		err = m.connections.Create(conn)
	}
	if err != nil {
		return nil, err
	}
	_ = m.states.Delete(st.ID)
	return conn, nil
}

func (m *Manager) GetConnection(platform string, chatID int, service string) (*models.ServiceConnection, error) {
	return m.GetConnectionContext(context.Background(), platform, chatID, service)
}

func (m *Manager) GetConnectionContext(ctx context.Context, platform string, chatID int, service string) (*models.ServiceConnection, error) {
	rows, err := m.connections.Where("platform = ? AND chat_id = ? AND service = ?", platform, chatID, service)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	conn := &rows[0]
	if service == LinearService && !conn.ExpiresAt.IsZero() && time.Until(conn.ExpiresAt) < time.Minute {
		if err := m.RefreshLinearConnection(ctx, conn); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (m *Manager) RefreshLinearConnection(ctx context.Context, conn *models.ServiceConnection) error {
	if conn.RefreshToken == "" {
		// Some Linear apps issue long-lived tokens without refresh tokens.
		conn.ExpiresAt = time.Time{}
		return m.connections.Update(conn)
	}
	oauthConfig, err := m.linearOAuthConfig()
	if err != nil {
		return err
	}
	tok, err := oauthConfig.Refresh(ctx, conn.RefreshToken)
	if err != nil {
		return err
	}
	conn.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		conn.RefreshToken = tok.RefreshToken
	}
	conn.ExpiresAt = tok.ExpiresAt
	return m.connections.Update(conn)
}

func ChatID(s string) int { i, _ := strconv.Atoi(s); return i }
