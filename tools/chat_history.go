package tools

import (
	"context"
	"encoding/json"
	"sort"

	"charm.land/fantasy"
	"github.com/rbrick/clanker/chat"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/summarize"
)

type ChatHistoryTool struct {
	history    *chat.ChatHistory
	summarizer *summarize.Service
}

func (t *ChatHistoryTool) Tools() []fantasy.AgentTool {
	type ChatHistoryInput struct {
		Platform string `json:"platform" jsonschema:"description=Platform name, e.g. telegram"`
		ChatID   int    `json:"chat_id"`
		User     string `json:"user" jsonschema:"description=Optional sender username, display name, or numeric user ID to filter messages by"`
		Limit    int    `json:"limit" jsonschema:"description=Maximum number of most recent messages to return; default 50"`
	}

	return []fantasy.AgentTool{
		fantasy.NewAgentTool[ChatHistoryInput](
			"get_chat_history",
			"get recent messages from a chat history by platform and chat_id; optionally filter to one sender by username, display name, or user ID",
			func(ctx context.Context, input ChatHistoryInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				messages, err := t.recentMessages(input.Platform, input.ChatID, input.User, input.Limit)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				jsonResponse, err := json.Marshal(messages)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				return fantasy.NewTextResponse(string(jsonResponse)), nil
			},
		),
		fantasy.NewAgentTool[ChatHistoryInput](
			"summarize_chat_history",
			"summarize recent messages from a chat history by platform and chat_id; set user to summarize what a specific person said",
			func(ctx context.Context, input ChatHistoryInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				messages, err := t.recentMessages(input.Platform, input.ChatID, input.User, input.Limit)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				summary, err := t.summarizer.Summarize(ctx, messages)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}
				return fantasy.NewTextResponse(summary), nil
			},
		),
	}
}

func (t *ChatHistoryTool) recentMessages(platform string, chatID int, user string, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	var messages []models.ChatMessage
	var err error
	if user != "" {
		messages, err = t.history.GetMessagesByUser(platform, chatID, user)
	} else {
		messages, err = t.history.GetMessages(platform, chatID)
	}
	if err != nil {
		return nil, err
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp > messages[j].Timestamp
	})

	if len(messages) > limit {
		messages = messages[:limit]
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	return messages, nil
}

func NewChatHistoryTool(history *chat.ChatHistory, summarizer *summarize.Service) *ChatHistoryTool {
	return &ChatHistoryTool{history: history, summarizer: summarizer}
}
