package summarize

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/database/models"
)

const systemPrompt = `You summarize chat history for Clanker.
Keep summaries concise, factual, and useful for answering follow-up questions.
Preserve names, decisions, TODOs, preferences, and unresolved questions.`

type Service struct {
	model fantasy.LanguageModel
}

func NewService(model fantasy.LanguageModel) *Service {
	return &Service{model: model}
}

func (s *Service) Summarize(ctx context.Context, messages []models.ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "No chat history found.", nil
	}

	var b strings.Builder
	for _, msg := range messages {
		name := msg.SenderUsername
		if msg.SenderName != "" {
			name = msg.SenderName
		}
		fmt.Fprintf(&b, "[%d] %s @%s (%d): %s\n", msg.Timestamp, name, msg.SenderUsername, msg.SenderID, msg.Message)
	}

	resp, err := s.model.Generate(ctx, fantasy.Call{
		Prompt: fantasy.Prompt{
			{
				Role: fantasy.MessageRoleSystem,
				Content: []fantasy.MessagePart{
					fantasy.TextPart{Text: systemPrompt},
				},
			},
			{
				Role: fantasy.MessageRoleUser,
				Content: []fantasy.MessagePart{
					fantasy.TextPart{Text: "Summarize this chat history:\n\n" + b.String()},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Content.Text(), nil
}
