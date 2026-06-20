package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/services"
)

type LinearTool struct{ services *services.Manager }

func NewLinearTool(m *services.Manager) *LinearTool { return &LinearTool{services: m} }

type linearGraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (t *LinearTool) linearGraphQL(ctx context.Context, platform string, chatID int, query string, vars map[string]interface{}, out interface{}) (string, error) {
	conn, err := t.services.GetConnectionContext(ctx, platform, chatID, services.LinearService)
	if err != nil {
		return "Linear is not connected for this chat. Ask an admin/user to run /connect and choose Linear.", nil
	}
	body, _ := json.Marshal(map[string]interface{}{"query": query, "variables": vars})
	doRequest := func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+conn.AccessToken)
		return http.DefaultClient.Do(req)
	}
	resp, err := doRequest()
	if err != nil {
		return err.Error(), err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := t.services.RefreshLinearConnection(ctx, conn); err != nil {
			return "Linear authorization expired and refresh failed; please run /connect and choose Linear again.", nil
		}
		resp, err = doRequest()
		if err != nil {
			return err.Error(), err
		}
	}
	defer resp.Body.Close()
	var gql linearGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gql); err != nil {
		return err.Error(), err
	}
	if resp.StatusCode >= 300 || len(gql.Errors) > 0 {
		msg := resp.Status
		if len(gql.Errors) > 0 {
			msg = gql.Errors[0].Message
		}
		return "Linear request failed: " + msg, nil
	}
	if out != nil {
		if err := json.Unmarshal(gql.Data, out); err != nil {
			return err.Error(), err
		}
	}
	return "", nil
}

func (t *LinearTool) Tools() []fantasy.AgentTool {
	type Input struct {
		Platform    string `json:"platform"`
		ChatID      int    `json:"chat_id"`
		TeamID      string `json:"team_id"`
		ProjectID   string `json:"project_id,omitempty"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    int    `json:"priority,omitempty"`
	}
	type ChatInput struct {
		Platform string `json:"platform"`
		ChatID   int    `json:"chat_id"`
	}
	type ProjectsInput struct {
		Platform string `json:"platform"`
		ChatID   int    `json:"chat_id"`
		TeamID   string `json:"team_id,omitempty"`
	}
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[ChatInput]("get_linear_context", "get a concise overview of the Linear workspace available to this chat, including teams and projects with IDs, keys/names, states, URLs, and project team associations. Use this first when a user asks to create a Linear ticket and refers to a team or project by name/key so you understand what is available.", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			var out struct {
				Teams struct {
					Nodes []struct{ ID, Key, Name string } `json:"nodes"`
				} `json:"teams"`
				Projects struct {
					Nodes []struct {
						ID, Name, URL, State string
						Teams                struct {
							Nodes []struct{ ID, Key, Name string } `json:"nodes"`
						} `json:"teams"`
					} `json:"nodes"`
				} `json:"projects"`
			}
			query := "query { teams { nodes { id key name } } projects { nodes { id name url state teams { nodes { id key name } } } } }"
			msg, err := t.linearGraphQL(ctx, input.Platform, input.ChatID, query, nil, &out)
			if msg != "" || err != nil {
				return fantasy.NewTextResponse(msg), err
			}
			b, _ := json.Marshal(out)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[ChatInput]("list_linear_teams", "list Linear teams available to this chat connection with IDs, keys, and names; use this to resolve user-provided team names/keys to team_id before creating tickets", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			var out struct {
				Teams struct {
					Nodes []struct{ ID, Key, Name string } `json:"nodes"`
				} `json:"teams"`
			}
			msg, err := t.linearGraphQL(ctx, input.Platform, input.ChatID, "query { teams { nodes { id key name } } }", nil, &out)
			if msg != "" || err != nil {
				return fantasy.NewTextResponse(msg), err
			}
			b, _ := json.Marshal(out.Teams.Nodes)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[ProjectsInput]("list_linear_projects", "list Linear projects available to this chat connection with IDs, names, URLs, states, and associated teams; optionally pass team_id to filter projects for a team. Use this to resolve user-provided project names to project_id before creating tickets in a project", func(ctx context.Context, input ProjectsInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			query := "query { projects { nodes { id name url state teams { nodes { id key name } } } } }"
			vars := map[string]interface{}(nil)
			if input.TeamID != "" {
				query = "query Projects($teamId: ID!) { team(id: $teamId) { projects { nodes { id name url state teams { nodes { id key name } } } } } }"
				vars = map[string]interface{}{"teamId": input.TeamID}
			}
			var out struct {
				Projects struct {
					Nodes []struct {
						ID, Name, URL, State string
						Teams                struct {
							Nodes []struct{ ID, Key, Name string } `json:"nodes"`
						} `json:"teams"`
					} `json:"nodes"`
				} `json:"projects"`
				Team struct {
					Projects struct {
						Nodes []struct {
							ID, Name, URL, State string
							Teams                struct {
								Nodes []struct{ ID, Key, Name string } `json:"nodes"`
							} `json:"teams"`
						} `json:"nodes"`
					} `json:"projects"`
				} `json:"team"`
			}
			msg, err := t.linearGraphQL(ctx, input.Platform, input.ChatID, query, vars, &out)
			if msg != "" || err != nil {
				return fantasy.NewTextResponse(msg), err
			}
			projects := out.Projects.Nodes
			if input.TeamID != "" {
				projects = out.Team.Projects.Nodes
			}
			b, _ := json.Marshal(projects)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[Input]("create_linear_ticket", "create a Linear issue for the current chat. Requires platform and chat_id from the message, a Linear team_id, title, optional description, optional project_id, and priority (0 none, 1 urgent, 2 high, 3 normal, 4 low). The title must be only the task name; do not include routing phrases like 'under project ...', 'in project ...', team names, or project names in the title when those are provided separately as team_id/project_id. If not connected, tell the user to run /connect and choose Linear.", func(ctx context.Context, input Input, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.TeamID == "" || input.Title == "" {
				return fantasy.NewTextResponse("team_id and title are required"), nil
			}
			vars := map[string]interface{}{"input": map[string]interface{}{"teamId": input.TeamID, "title": input.Title, "description": input.Description}}
			if input.ProjectID != "" {
				vars["input"].(map[string]interface{})["projectId"] = input.ProjectID
			}
			if input.Priority != 0 {
				vars["input"].(map[string]interface{})["priority"] = input.Priority
			}
			var out struct {
				IssueCreate struct {
					Issue struct{ Identifier, URL string } `json:"issue"`
				} `json:"issueCreate"`
			}
			msg, err := t.linearGraphQL(ctx, input.Platform, input.ChatID, "mutation IssueCreate($input: IssueCreateInput!) { issueCreate(input: $input) { success issue { identifier title url } } }", vars, &out)
			if msg != "" || err != nil {
				return fantasy.NewTextResponse(msg), err
			}
			issue := out.IssueCreate.Issue
			return fantasy.NewTextResponse(fmt.Sprintf("Created Linear issue %s: %s", issue.Identifier, issue.URL)), nil
		}),
	}
}
