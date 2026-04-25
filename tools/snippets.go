package tools

import (
	"context"
	"encoding/json"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/snippets"
)

type SnippetsTool struct {
	snippets *snippets.Snippets
}

func (s *SnippetsTool) Tools() []fantasy.AgentTool {
	type SnippetsToolInput struct {
		Content  string `json:"content"`
		Language string `json:"language"`
	}

	type GetSnippetByIDInput struct {
		ID int `json:"id"`
	}
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[SnippetsToolInput](

			"create_snippet",
			"create a code snippet with the given content and programming language",
			func(ctx context.Context, input SnippetsToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				snippet, err := s.snippets.CreateSnippet(input.Content, input.Language)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				jsonResponse, err := json.Marshal(snippet)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				return fantasy.NewTextResponse(string(jsonResponse)), nil
			},
		),

		fantasy.NewAgentTool[GetSnippetByIDInput](
			"get_snippet_by_id",
			"get a code snippet by its ID",
			func(ctx context.Context, input GetSnippetByIDInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				snippet, err := s.snippets.GetSnippetByID(input.ID)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				if snippet == nil {
					return fantasy.NewTextResponse("snippet not found"), nil
				}
				jsonResponse, err := json.Marshal(snippet)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				return fantasy.NewTextResponse(string(jsonResponse)), nil
			},
		),
	}
}

func NewSnippetsTool(snippets *snippets.Snippets) *SnippetsTool {
	return &SnippetsTool{snippets: snippets}
}
