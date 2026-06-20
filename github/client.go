package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultAPIBaseURL = "https://api.github.com/"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(opts ...Option) *Client {
	c := &Client{baseURL: DefaultAPIBaseURL, httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = baseURL
		}
	}
}

type User struct {
	Name              *string `json:"name"`
	Email             *string `json:"email"`
	Login             string  `json:"login"`
	ID                int64   `json:"id"`
	NodeID            string  `json:"node_id"`
	AvatarURL         string  `json:"avatar_url"`
	GravatarID        *string `json:"gravatar_id"`
	URL               string  `json:"url"`
	HTMLURL           string  `json:"html_url"`
	FollowersURL      string  `json:"followers_url"`
	FollowingURL      string  `json:"following_url"`
	GistsURL          string  `json:"gists_url"`
	StarredURL        string  `json:"starred_url"`
	SubscriptionsURL  string  `json:"subscriptions_url"`
	OrganizationsURL  string  `json:"organizations_url"`
	ReposURL          string  `json:"repos_url"`
	EventsURL         string  `json:"events_url"`
	ReceivedEventsURL string  `json:"received_events_url"`
	Type              string  `json:"type"`
	SiteAdmin         bool    `json:"site_admin"`
	StarredAt         string  `json:"starred_at"`
	UserViewType      string  `json:"user_view_type"`
}

type GistFile struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
	Encoding  string `json:"encoding"`
}

type GistForkOfFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
	Size     int    `json:"size"`
}

type Gist struct {
	ForkOf          *Gist                      `json:"fork_of"`
	URL             string                     `json:"url"`
	ForksURL        string                     `json:"forks_url"`
	CommitsURL      string                     `json:"commits_url"`
	ID              string                     `json:"id"`
	NodeID          string                     `json:"node_id"`
	GitPullURL      string                     `json:"git_pull_url"`
	GitPushURL      string                     `json:"git_push_url"`
	HTMLURL         string                     `json:"html_url"`
	Files           map[string]*GistFile       `json:"files"`
	Public          bool                       `json:"public"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	Description     *string                    `json:"description"`
	Comments        int                        `json:"comments"`
	CommentsEnabled bool                       `json:"comments_enabled"`
	User            *string                    `json:"user"`
	CommentsURL     string                     `json:"comments_url"`
	Owner           *User                      `json:"owner"`
	Truncated       bool                       `json:"truncated"`
	Forks           []json.RawMessage          `json:"forks"`
	History         []json.RawMessage          `json:"history"`
	ForkFiles       map[string]*GistForkOfFile `json:"-"`
}

func (c *Client) Gist(ctx context.Context, gistID string) (*Gist, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"gists/"+gistID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github gist request failed: %s", resp.Status)
	}
	var gist Gist
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return nil, err
	}
	return &gist, nil
}
