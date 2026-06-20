package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ImagesEndpoint = "https://api.openai.com/v1/images/generations"

type ImageClient struct {
	apiKey     string
	httpClient *http.Client
	endpoint   string
}

func NewImageClient(apiKey string, opts ...ImageOption) *ImageClient {
	c := &ImageClient{apiKey: apiKey, httpClient: &http.Client{Timeout: 2 * time.Minute}, endpoint: ImagesEndpoint}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type ImageOption func(*ImageClient)

func WithImageHTTPClient(httpClient *http.Client) ImageOption {
	return func(c *ImageClient) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

type ImageRequest struct {
	Model  string
	Prompt string
	Size   string
}

func (c *ImageClient) Generate(ctx context.Context, input ImageRequest) ([]byte, error) {
	requestBody := map[string]any{"model": input.Model, "prompt": input.Prompt, "size": input.Size}
	if input.Model == "dall-e-2" || input.Model == "dall-e-3" {
		requestBody["response_format"] = "b64_json"
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image generation failed: %s", string(respBody))
	}
	var parsed struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("image generation returned no image data")
	}
	if parsed.Data[0].B64JSON != "" {
		return base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	}
	if parsed.Data[0].URL != "" {
		return c.download(ctx, parsed.Data[0].URL)
	}
	return nil, fmt.Errorf("image generation returned no image data")
}

func (c *ImageClient) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image download failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
