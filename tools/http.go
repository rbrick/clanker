package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"charm.land/fantasy"
)

func HTTPTool() fantasy.AgentTool {
	type HTTPToolInput struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
	}
	return fantasy.NewAgentTool[HTTPToolInput](
		"http_request",
		"make HTTP requests to interact with web services and APIs",
		func(ctx context.Context, input HTTPToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			method := strings.ToUpper(input.Method)
			if method == "" {
				method = http.MethodGet
			}
			log.Printf("calling http tool method=%s url=%s", method, input.URL)

			var body io.Reader
			if input.Body != "" {
				body = bytes.NewReader([]byte(input.Body))
			}
			req, err := http.NewRequestWithContext(ctx, method, input.URL, body)
			if err != nil {
				return fantasy.NewTextResponse(err.Error()), err
			}
			for key, value := range input.Headers {
				req.Header.Set(key, value)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fantasy.NewTextResponse(err.Error()), err
			}
			defer resp.Body.Close()

			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fantasy.NewTextResponse(err.Error()), err
			}
			return fantasy.NewTextResponse(fmt.Sprintf("HTTP %s\n%s", resp.Status, responseBody)), nil
		},
	)
}
