package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/linear"
	"github.com/rbrick/clanker/services"
)

type LinearTool struct{ services *services.Manager }

func NewLinearTool(m *services.Manager) *LinearTool { return &LinearTool{services: m} }

func withLinearClient[T any](ctx context.Context, t *LinearTool, platform string, chatID int, fn func(*linear.Client) (T, error)) (T, error) {
	var zero T
	conn, err := t.services.GetConnectionContext(ctx, platform, chatID, services.LinearService)
	if err != nil {
		return zero, errNotConnected
	}
	client := linear.NewClient(conn.AccessToken)
	out, err := fn(client)
	if !errors.Is(err, linear.ErrUnauthorized) {
		return out, err
	}
	if err := t.services.RefreshLinearConnection(ctx, conn); err != nil {
		return zero, errAuthExpired
	}
	client.SetAccessToken(conn.AccessToken)
	return fn(client)
}

var (
	errNotConnected = errors.New("linear is not connected")
	errAuthExpired  = errors.New("linear authorization expired")
)

func linearErrorResponse(err error) (fantasy.ToolResponse, error) {
	switch {
	case errors.Is(err, errNotConnected):
		return fantasy.NewTextResponse("Linear is not connected for this chat. Ask an admin/user to run /connect and choose Linear."), nil
	case errors.Is(err, errAuthExpired):
		return fantasy.NewTextResponse("Linear authorization expired and refresh failed; please run /connect and choose Linear again."), nil
	default:
		return fantasy.NewTextResponse("Linear request failed: " + err.Error()), nil
	}
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
		AssigneeID  string `json:"assignee_id,omitempty"`
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
		fantasy.NewAgentTool[ChatInput]("get_linear_context", "get a concise overview of the Linear workspace available to this chat, including teams, projects, and users with IDs. Use this first when a user asks to create a Linear ticket and refers to a team, project, or assignee by name/key/email so you understand what is available.", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			out, err := withLinearClient(ctx, t, input.Platform, input.ChatID, func(client *linear.Client) (*linear.WorkspaceContext, error) {
				return client.Context(ctx)
			})
			if err != nil {
				return linearErrorResponse(err)
			}
			b, _ := json.Marshal(out)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[ChatInput]("list_linear_teams", "list Linear teams available to this chat connection with IDs, keys, and names; use this to resolve user-provided team names/keys to team_id before creating tickets", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			teams, err := withLinearClient(ctx, t, input.Platform, input.ChatID, func(client *linear.Client) ([]linear.Team, error) {
				return client.Teams(ctx)
			})
			if err != nil {
				return linearErrorResponse(err)
			}
			b, _ := json.Marshal(teams)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[ProjectsInput]("list_linear_projects", "list Linear projects available to this chat connection with IDs, names, URLs, states, and associated teams; optionally pass team_id to filter projects for a team. Use this to resolve user-provided project names to project_id before creating tickets in a project", func(ctx context.Context, input ProjectsInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			projects, err := withLinearClient(ctx, t, input.Platform, input.ChatID, func(client *linear.Client) ([]linear.Project, error) {
				return client.Projects(ctx, input.TeamID)
			})
			if err != nil {
				return linearErrorResponse(err)
			}
			b, _ := json.Marshal(projects)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[ChatInput]("list_linear_users", "list Linear users available to this chat connection with IDs, names, display names, and emails; use this to resolve user-provided assignee names/emails to assignee_id before creating tickets", func(ctx context.Context, input ChatInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			users, err := withLinearClient(ctx, t, input.Platform, input.ChatID, func(client *linear.Client) ([]linear.User, error) {
				return client.Users(ctx)
			})
			if err != nil {
				return linearErrorResponse(err)
			}
			b, _ := json.Marshal(users)
			return fantasy.NewTextResponse(string(b)), nil
		}),
		fantasy.NewAgentTool[Input]("create_linear_ticket", "create a Linear issue for the current chat. Requires platform and chat_id from the message, a Linear team_id, title, optional description, optional project_id, optional assignee_id, and priority (0 none, 1 urgent, 2 high, 3 normal, 4 low). Resolve mentioned assignees with get_linear_context or list_linear_users first and pass assignee_id. The title must be only the task name; do not include routing phrases like 'under project ...', 'in project ...', team names, project names, or assignee names in the title when those are provided separately as team_id/project_id/assignee_id. If not connected, tell the user to run /connect and choose Linear.", func(ctx context.Context, input Input, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.TeamID == "" || input.Title == "" {
				return fantasy.NewTextResponse("team_id and title are required"), nil
			}
			issue, err := withLinearClient(ctx, t, input.Platform, input.ChatID, func(client *linear.Client) (*linear.Issue, error) {
				return client.CreateIssue(ctx, linear.IssueCreateInput{
					TeamID:      input.TeamID,
					Title:       input.Title,
					Description: input.Description,
					ProjectID:   input.ProjectID,
					AssigneeID:  input.AssigneeID,
					Priority:    input.Priority,
				})
			})
			if err != nil {
				return linearErrorResponse(err)
			}
			return fantasy.NewTextResponse(fmt.Sprintf("Created Linear issue %s: %s", issue.Identifier, issue.URL)), nil
		}),
	}
}
