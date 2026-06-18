package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

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
					log.Printf("image generation failed: %v", err)
					return fantasy.NewTextResponse("image generation failed: " + err.Error()), nil
				}
				out, _ := json.Marshal(map[string]string{"image_url": url})
				return fantasy.NewTextResponse(string(out)), nil
			},
		),
	}
}

func (t *ImageGeneratorTool) generate(ctx context.Context, input ImageGeneratorInput) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

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

	model := os.Getenv("IMAGE_MODEL")
	if model == "" {
		model = "dall-e-3"
	}

	requestBody := map[string]any{
		"model":           model,
		"prompt":          input.Prompt,
		"size":            input.Size,
		"response_format": "b64_json",
	}
	if model == "gpt-image-1" {
		// gpt-image-1 returns b64_json by default and rejects response_format.
		delete(requestBody, "response_format")
	}
	body, _ := json.Marshal(requestBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("generating image with model=%s size=%s", model, input.Size)
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
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
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Data) == 0 {
		return "", fmt.Errorf("image generation returned no image data")
	}

	var data []byte
	if parsed.Data[0].B64JSON != "" {
		data, err = base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
		if err != nil {
			return "", err
		}
	} else if parsed.Data[0].URL != "" {
		data, err = downloadImage(ctx, client, parsed.Data[0].URL)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("image generation returned no image data")
	}
	blob, err := t.store.Save("image/png", data)
	if err != nil {
		return "", err
	}
	url := media.PublicURL(t.baseURL, blob.ID)
	log.Printf("generated image stored as %s", url)
	return url, nil
}

func downloadImage(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image download failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
