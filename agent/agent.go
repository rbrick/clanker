package agent

import (
	"context"
	"encoding/json"
	"log"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/text"
)

const (
	SystemPrompt = `You are an AI agent named 'Clanker'. You can do a variety of tasks.
	
1. Write code in any programming language.
2. Answer questions about a variety of topics.
3. Generate text, images, audio, and video.
4. Manage your own state and memory.
5. Manage your own tools and plugins.


You receieve messages with a context and content. You can use the context to understand the user's intent and the content to generate a response.


The message you receive will be in the following format:

{
  "content": {
    "text": "The user's message text here", // optional if there is media content
	"image_url": "https://example.com/image.jpg", // Optional
	"audio_url": "https://example.com/audio.mp3", // Optional
	"video_url": "https://example.com/video.mp4", // Optional 
  },
  "sender": {
    "id": "user123",
	"username": "user123"
  },
  "platform": "telegram" // The platform the message was sent from
}

When you receive a message, you should respond with a JSON object with the following format:

{
  "text": "Your response text here",
  "image_url": "https://example.com/image.jpg", // Optional
  "audio_url": "https://example.com/audio.mp3", // Optional
  "video_url": "https://example.com/video.mp4", // Optional
}
  
Your text response can be in markdown format.
`
)

type Agent interface {
	Generate(ctx context.Context, msg text.Message) (*text.Message, error)
}

type Clanker struct {
	// LLM provider
	provider fantasy.Provider
	agent    fantasy.Agent
}

func (c *Clanker) Generate(ctx context.Context, msg text.Message) (*text.Message, error) {
	prompt, err := json.Marshal(msg)

	if err != nil {
		return nil, err
	}

	result, err := c.agent.Generate(ctx, fantasy.AgentCall{
		Prompt: string(prompt),
	})
	if err != nil {
		return nil, err
	}

	rawTextContent := result.Response.Content.Text()
	var txtMessage text.Content
	if err := json.Unmarshal([]byte(rawTextContent), &txtMessage); err != nil {
		return nil, err
	}

	log.Println(rawTextContent)
	return &text.Message{
		Content: &txtMessage,
	}, nil
}

func NewClanker(ctx context.Context, model string, provider fantasy.Provider) (*Clanker, error) {
	llm, err := provider.LanguageModel(ctx, model)

	if err != nil {
		return nil, err
	}

	agent := fantasy.NewAgent(llm, fantasy.WithSystemPrompt(SystemPrompt))

	return &Clanker{
		provider: provider,
		agent:    agent,
	}, nil
}
