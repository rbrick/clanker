package tools

import (
	"context"
	"encoding/json"

	"charm.land/fantasy"
	githubclient "github.com/rbrick/clanker/github"
)

type GithubTool struct{ client *githubclient.Client }

func NewGithubTool() *GithubTool { return &GithubTool{client: githubclient.NewClient()} }

func (g *GithubTool) Tools() []fantasy.AgentTool {
	type GithubToolInput struct {
		GistID string `json:"gist_id"`
	}

	return []fantasy.AgentTool{
		fantasy.NewAgentTool[GithubToolInput](
			"read_gist",
			"read a GitHub gist by gist ID",
			func(ctx context.Context, input GithubToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				gist, err := g.client.Gist(ctx, input.GistID)
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
