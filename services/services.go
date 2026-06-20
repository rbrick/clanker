package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
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
	callback := stringsTrimRight(m.publicURL, "/") + "/oauth/linear/callback"
	u, _ := url.Parse("https://linear.app/oauth/authorize")
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", callback)
	q.Set("response_type", "code")
	q.Set("scope", "read,write")
	q.Set("state", state)
	q.Set("prompt", "consent")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func stringsTrimRight(s, cutset string) string {
	for len(s) > 0 && cutset == "/" && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
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
	clientID, secret := os.Getenv("LINEAR_CLIENT_ID"), os.Getenv("LINEAR_CLIENT_SECRET")
	if clientID == "" || secret == "" {
		return nil, errors.New("LINEAR_CLIENT_ID/LINEAR_CLIENT_SECRET are not configured")
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {stringsTrimRight(m.publicURL, "/") + "/oauth/linear/callback"}, "client_id": {clientID}, "client_secret": {secret}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/oauth/token", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 || tok.AccessToken == "" {
		return nil, fmt.Errorf("linear token exchange failed: %s", resp.Status)
	}
	expiresAt := time.Time{}
	if tok.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	conn := &models.ServiceConnection{Platform: st.Platform, ChatID: st.ChatID, Service: LinearService, AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, ExpiresAt: expiresAt}
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
	clientID, secret := os.Getenv("LINEAR_CLIENT_ID"), os.Getenv("LINEAR_CLIENT_SECRET")
	if clientID == "" || secret == "" {
		return errors.New("LINEAR_CLIENT_ID/LINEAR_CLIENT_SECRET are not configured")
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {conn.RefreshToken}, "client_id": {clientID}, "client_secret": {secret}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/oauth/token", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return err
	}
	if resp.StatusCode >= 300 || tok.AccessToken == "" {
		return fmt.Errorf("linear token refresh failed: %s", resp.Status)
	}
	conn.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		conn.RefreshToken = tok.RefreshToken
	}
	conn.ExpiresAt = time.Time{}
	if tok.ExpiresIn > 0 {
		conn.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	return m.connections.Update(conn)
}

func ChatID(s string) int { i, _ := strconv.Atoi(s); return i }
