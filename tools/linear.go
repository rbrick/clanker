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
	conn, err := t.services.GetConnection(platform, chatID, services.LinearService)
	if err != nil {
		return "Linear is not connected for this chat. Ask an admin/user to run /connect and choose Linear.", nil
	}
	body, _ := json.Marshal(map[string]interface{}{"query": query, "variables": vars})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+conn.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err.Error(), err
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
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    int    `json:"priority,omitempty"`
	}
	type ChatInput struct {
		Platform string `json:"platform"`
		ChatID   int    `json:"chat_id"`
	}
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[ChatInput]("list_linear_teams", "list Linear teams available to this chat connection; use this to find a team_id before creating tickets", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
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
		fantasy.NewAgentTool[Input]("create_linear_ticket", "create a Linear issue for the current chat. Requires platform and chat_id from the message, a Linear team_id, title, optional description and priority (0 none, 1 urgent, 2 high, 3 normal, 4 low). If not connected, tell the user to run /connect and choose Linear.", func(ctx context.Context, input Input, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.TeamID == "" || input.Title == "" {
				return fantasy.NewTextResponse("team_id and title are required"), nil
			}
			vars := map[string]interface{}{"input": map[string]interface{}{"teamId": input.TeamID, "title": input.Title, "description": input.Description}}
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
