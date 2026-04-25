package tools

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"

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

			log.Println("calling http tool", input)
			req, err := http.NewRequest(input.Method, input.URL, nil)
			if err != nil {
				return fantasy.ToolResponse{
					Type:    "error",
					Content: err.Error(),
				}, err
			}

			for key, value := range input.Headers {
				// Add headers to the request
				req.Header.Add(key, value)
			}

			if input.Body != "" {
				req.Body = io.NopCloser(bytes.NewReader([]byte(input.Body)))
			}

			response, err := http.DefaultClient.Do(req)
			log.Println(response)
			if err != nil {
				return fantasy.NewTextResponse(err.Error()), err
			}

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			if err != nil {
				return fantasy.NewTextResponse(err.Error()), err
			}

			log.Println(string(responseBody))
			return fantasy.NewTextResponse(string(responseBody)), nil
		},
	)
}
