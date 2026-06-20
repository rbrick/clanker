package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/media"
	"github.com/rbrick/clanker/openai"
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

	apiKey := firstEnv("OPENAI_API_KEY", "LLM_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY or LLM_API_KEY is required for image generation")
	}
	if input.Size == "" {
		input.Size = "1024x1024"
	}
	model := os.Getenv("IMAGE_MODEL")
	if model == "" {
		model = "gpt-image-1"
	}

	log.Printf("generating image with model=%s size=%s", model, input.Size)
	data, err := openai.NewImageClient(apiKey).Generate(ctx, openai.ImageRequest{Model: model, Prompt: input.Prompt, Size: input.Size})
	if err != nil {
		return "", err
	}
	blob, err := t.store.Save("image/png", data)
	if err != nil {
		return "", err
	}
	url := media.PublicURL(t.baseURL, blob.ID)
	log.Printf("generated image stored as %s", url)
	return url, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
