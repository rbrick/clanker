package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/media"
)

type ImageGeneratorInput struct {
	Prompt string `json:"prompt" jsonschema:"description=Detailed prompt describing the image to generate"`
	Size   string `json:"size" jsonschema:"description=Image size, e.g. 1024x1024, 1024x1536, or 1536x1024"`
}

type ImageGeneratorTool struct {
	store   *media.Store
	baseURL string
}

func NewImageGeneratorTool(store *media.Store, baseURL string) *ImageGeneratorTool {
	return &ImageGeneratorTool{store: store, baseURL: baseURL}
}

func (t *ImageGeneratorTool) Tools() []fantasy.AgentTool {
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[ImageGeneratorInput](
			"generate_image",
			"Generate an image from a prompt, store it as a SQL blob, and return a public image_url. Use this whenever the user asks for an image, picture, drawing, logo, meme, etc.",
			func(ctx context.Context, input ImageGeneratorInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				url, err := t.generate(ctx, input)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				out, _ := json.Marshal(map[string]string{"image_url": url})
				return fantasy.NewTextResponse(string(out)), nil
			},
		),
	}
}

func (t *ImageGeneratorTool) generate(ctx context.Context, input ImageGeneratorInput) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY or LLM_API_KEY is required for image generation")
	}
	if input.Size == "" {
		input.Size = "1024x1024"
	}

	body, _ := json.Marshal(map[string]any{
		"model":           os.Getenv("IMAGE_MODEL"),
		"prompt":          input.Prompt,
		"size":            input.Size,
		"response_format": "b64_json",
	})
	if os.Getenv("IMAGE_MODEL") == "" {
		body, _ = json.Marshal(map[string]any{"model": "gpt-image-1", "prompt": input.Prompt, "size": input.Size})
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("image generation failed: %s", string(respBody))
	}

	var parsed struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Data) == 0 || parsed.Data[0].B64JSON == "" {
		return "", fmt.Errorf("image generation returned no image data")
	}
	data, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return "", err
	}
	blob, err := t.store.Save("image/png", data)
	if err != nil {
		return "", err
	}
	return media.PublicURL(t.baseURL, blob.ID), nil
}
