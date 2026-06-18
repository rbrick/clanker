package agent

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/text"
	"github.com/rbrick/clanker/tools"
)

const (
	SystemPrompt = `You are an AI agent named 'Clanker'. You can do a variety of tasks.
	
1. Write code in any programming language. For coding requests, act like a coding agent: give complete, runnable files/commands and sensible defaults instead of doing nothing or asking unnecessary questions. If the user asks for something like a Bukkit plugin, provide a minimal project structure plus source code and build instructions.
2. Answer questions about a variety of topics.
3. Generate text, images, audio, and video.
4. Manage your own state and memory.
5. Manage your own tools and plugins.

You receive messages with context and content. Always use the context to understand the user's intent and the content to generate a response. The context contains recent messages in the chat, oldest first, including replies when available. If the current message is a reply, use reply_to_message_id and the recent context to understand what it refers to.


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
	"username": "user123",
	"name": "User Display Name"
  },
  "chat": {
    "id": "12345",
	"type": "group"
  },
  "platform": "telegram", // The platform the message was sent from
  "reply_to_message_id": "123", // Optional: platform message ID this message replies to
  "context": [ // Recent chat messages, oldest first
    {
      "id": "120",
      "sender": {"id":"user456","username":"alice","name":"Alice"},
      "content": {"text":"Earlier message"},
      "reply_to_message_id": "119"
    }
  ]
}

For requests like "summarize what Alice said" or "what did @bob say", use the summarize_chat_history tool with the current message's platform and chat.id, and pass the requested person as user.

For requests to create/generate/draw an image, use the generate_image tool. Put the returned URL in your final JSON response as image_url, and include only a short caption in text.

When you receive a message, you must respond with only a JSON object with the following format. Do not wrap the JSON in markdown fences. Escape newlines in strings as needed.

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

	maxOutputTokens := int64(8192)
	result, err := c.agent.Generate(ctx, fantasy.AgentCall{
		Prompt:          string(prompt),
		MaxOutputTokens: &maxOutputTokens,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	rawTextContent := result.Response.Content.Text()
	var txtMessage text.Content
	if err := json.Unmarshal([]byte(rawTextContent), &txtMessage); err != nil {
		log.Printf("agent returned non-JSON response, falling back to raw text: %v", err)
		txtMessage.Text = cleanupRawResponse(rawTextContent)
	}

	return &text.Message{
		Content: &txtMessage,
	}, nil
}

func cleanupRawResponse(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
		var c text.Content
		if err := json.Unmarshal([]byte(s), &c); err == nil {
			return c.Text
		}
	}
	return s
}

func NewClanker(ctx context.Context, model string, provider fantasy.Provider, agentTools ...fantasy.AgentTool) (*Clanker, error) {
	llm, err := provider.LanguageModel(ctx, model)

	if err != nil {
		return nil, err
	}

	agentTools = append(agentTools, tools.MinecraftPingerTool(), tools.HTTPTool())

	agentTools = append(agentTools, tools.NewGithubTool().Tools()...)

	agent := fantasy.NewAgent(llm, fantasy.WithSystemPrompt(SystemPrompt), fantasy.WithTools(agentTools...))

	return &Clanker{
		provider: provider,
		agent:    agent,
	}, nil
}
