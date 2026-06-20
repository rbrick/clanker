package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const graphQLEndpoint = "https://api.linear.app/graphql"

var ErrUnauthorized = errors.New("linear: unauthorized")

type Client struct {
	accessToken string
	httpClient  *http.Client
	endpoint    string
}

func NewClient(accessToken string, opts ...Option) *Client {
	c := &Client{accessToken: accessToken, httpClient: http.DefaultClient, endpoint: graphQLEndpoint}
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

func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		if endpoint != "" {
			c.endpoint = endpoint
		}
	}
}

func (c *Client) SetAccessToken(accessToken string) { c.accessToken = accessToken }

type Team struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Project struct {
	ID    string           `json:"id"`
	Name  string           `json:"name"`
	URL   string           `json:"url"`
	State string           `json:"state"`
	Teams Connection[Team] `json:"teams"`
}

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type Issue struct {
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}

type Connection[T any] struct {
	Nodes []T `json:"nodes"`
}

type WorkspaceContext struct {
	Teams    Connection[Team]    `json:"teams"`
	Projects Connection[Project] `json:"projects"`
	Users    Connection[User]    `json:"users"`
}

type IssueCreateInput struct {
	TeamID      string `json:"teamId"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	AssigneeID  string `json:"assigneeId,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) graphQL(ctx context.Context, query string, vars map[string]interface{}, out interface{}) error {
	body, err := json.Marshal(map[string]interface{}{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	var gql graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gql); err != nil {
		return err
	}
	if resp.StatusCode >= 300 || len(gql.Errors) > 0 {
		msg := resp.Status
		if len(gql.Errors) > 0 {
			msg = gql.Errors[0].Message
		}
		return fmt.Errorf("linear request failed: %s", msg)
	}
	if out != nil {
		return json.Unmarshal(gql.Data, out)
	}
	return nil
}

func (c *Client) Context(ctx context.Context) (*WorkspaceContext, error) {
	var out WorkspaceContext
	query := "query { teams { nodes { id key name } } projects { nodes { id name url state teams { nodes { id key name } } } } users { nodes { id name displayName email } } }"
	if err := c.graphQL(ctx, query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Teams(ctx context.Context) ([]Team, error) {
	var out struct {
		Teams Connection[Team] `json:"teams"`
	}
	if err := c.graphQL(ctx, "query { teams { nodes { id key name } } }", nil, &out); err != nil {
		return nil, err
	}
	return out.Teams.Nodes, nil
}

func (c *Client) Projects(ctx context.Context, teamID string) ([]Project, error) {
	if teamID == "" {
		var out struct {
			Projects Connection[Project] `json:"projects"`
		}
		if err := c.graphQL(ctx, "query { projects { nodes { id name url state teams { nodes { id key name } } } } }", nil, &out); err != nil {
			return nil, err
		}
		return out.Projects.Nodes, nil
	}
	var out struct {
		Team struct {
			Projects Connection[Project] `json:"projects"`
		} `json:"team"`
	}
	query := "query Projects($teamId: ID!) { team(id: $teamId) { projects { nodes { id name url state teams { nodes { id key name } } } } } }"
	if err := c.graphQL(ctx, query, map[string]interface{}{"teamId": teamID}, &out); err != nil {
		return nil, err
	}
	return out.Team.Projects.Nodes, nil
}

func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out struct {
		Users Connection[User] `json:"users"`
	}
	if err := c.graphQL(ctx, "query { users { nodes { id name displayName email } } }", nil, &out); err != nil {
		return nil, err
	}
	return out.Users.Nodes, nil
}

func (c *Client) CreateIssue(ctx context.Context, input IssueCreateInput) (*Issue, error) {
	var out struct {
		IssueCreate struct {
			Issue Issue `json:"issue"`
		} `json:"issueCreate"`
	}
	query := "mutation IssueCreate($input: IssueCreateInput!) { issueCreate(input: $input) { success issue { identifier title url } } }"
	if err := c.graphQL(ctx, query, map[string]interface{}{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.IssueCreate.Issue, nil
}
