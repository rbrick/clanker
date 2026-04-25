package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"charm.land/fantasy"
)

const (
	GithubAPIUrlPrefix = "https://api.github.com/"
)

type GithubUser struct {
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

type GithubGistFile struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
	Encoding  string `json:"encoding"`
}

type GithubGistForkOfFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
	Size     int    `json:"size"`
}

type GithubGist struct {
	ForkOf          *GithubGist                      `json:"fork_of"`
	URL             string                           `json:"url"`
	ForksURL        string                           `json:"forks_url"`
	CommitsURL      string                           `json:"commits_url"`
	ID              string                           `json:"id"`
	NodeID          string                           `json:"node_id"`
	GitPullURL      string                           `json:"git_pull_url"`
	GitPushURL      string                           `json:"git_push_url"`
	HTMLURL         string                           `json:"html_url"`
	Files           map[string]*GithubGistFile       `json:"files"`
	Public          bool                             `json:"public"`
	CreatedAt       time.Time                        `json:"created_at"`
	UpdatedAt       time.Time                        `json:"updated_at"`
	Description     *string                          `json:"description"`
	Comments        int                              `json:"comments"`
	CommentsEnabled bool                             `json:"comments_enabled"`
	User            *string                          `json:"user"`
	CommentsURL     string                           `json:"comments_url"`
	Owner           *GithubUser                      `json:"owner"`
	Truncated       bool                             `json:"truncated"`
	Forks           []json.RawMessage                `json:"forks"`
	History         []json.RawMessage                `json:"history"`
	ForkFiles       map[string]*GithubGistForkOfFile `json:"-"`
}

type GithubTool struct{}

func (g *GithubTool) ReadGist(gistID string) (*GithubGist, error) {
	response, err := http.Get(fmt.Sprintf("%sgists/%s", GithubAPIUrlPrefix, gistID))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var gist GithubGist
	if err := json.Unmarshal(body, &gist); err != nil {
		return nil, err
	}

	return &gist, nil
}

func (g *GithubTool) Tools() []fantasy.AgentTool {
	type GithubToolInput struct {
		GistID string `json:"gist_id"`
	}

	return []fantasy.AgentTool{
		fantasy.NewAgentTool[GithubToolInput](
			"read_gist",
			"read a GitHub gist by gist ID",
			func(ctx context.Context, input GithubToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				gist, err := g.ReadGist(input.GistID)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				jsonResponse, err := json.Marshal(gist)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				return fantasy.NewTextResponse(string(jsonResponse)), nil
			},
		),
	}
}

func NewGithubTool() *GithubTool {
	return &GithubTool{}
}
