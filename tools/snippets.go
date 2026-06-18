package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"github.com/google/uuid"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/snippets"
)

type SnippetsTool struct {
	snippets *snippets.Snippets
}

func (s *SnippetsTool) Tools() []fantasy.AgentTool {
	type SnippetFileInput struct {
		Path     string `json:"path" jsonschema:"description=File path or display name"`
		Content  string `json:"content"`
		Language string `json:"language"`
	}
	type SnippetsToolInput struct {
		Content  string             `json:"content" jsonschema:"description=Single-file snippet content; ignored when files is provided"`
		Language string             `json:"language" jsonschema:"description=Single-file snippet language; ignored when files is provided"`
		Files    []SnippetFileInput `json:"files" jsonschema:"description=Multiple files for this snippet"`
	}
	type GetSnippetByIDInput struct {
		ID string `json:"id"`
	}
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[SnippetsToolInput](
			"create_snippet",
			"create a shareable code/project snippet. Supports either content/language for one file or files for multi-file snippets. The response includes a web URL that should be sent to the user. A git_url may also be present; if it is empty or unavailable, the web URL is still valid.",
			func(ctx context.Context, input SnippetsToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				files := make([]snippets.File, 0, len(input.Files))
				for _, file := range input.Files {
					files = append(files, snippets.File{Path: file.Path, Content: file.Content, Language: file.Language})
				}

				var snippet *models.Snippet
				var err error
				if len(files) > 0 {
					snippet, err = s.snippets.CreateSnippetWithFiles(files)
				} else {
					snippet, err = s.snippets.CreateSnippet(input.Content, input.Language)
				}
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				payload := struct {
					Snippet any    `json:"snippet"`
					URL     string `json:"url"`
				}{Snippet: snippet, URL: snippetURL(fmt.Sprint(snippet.ID))}

				jsonResponse, err := json.Marshal(payload)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				return fantasy.NewTextResponse(string(jsonResponse)), nil
			},
		),

		fantasy.NewAgentTool[GetSnippetByIDInput](
			"get_snippet_by_id",
			"get a code snippet by its UUID",
			func(ctx context.Context, input GetSnippetByIDInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				id, err := uuid.Parse(input.ID)
				if err != nil {
					return fantasy.NewTextResponse("invalid snippet id"), err
				}
				snippet, err := s.snippets.GetSnippetByID(id)
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

func snippetURL(id string) string {
	base := strings.TrimRight(os.Getenv("SNIPPET_BASE_URL"), "/")
	if base == "" {
		base = strings.TrimRight(os.Getenv("PUBLIC_WEB_URL"), "/")
	}
	if base == "" {
		return "/snippet/" + id
	}
	return base + "/snippet/" + id
}

func NewSnippetsTool(snippets *snippets.Snippets) *SnippetsTool {
	return &SnippetsTool{snippets: snippets}
}
