package chat

import (
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
)

type ChatHistory struct {
	repo database.Repository[models.ChatMessage]
}

func (h *ChatHistory) SaveMessage(msg *models.ChatMessage) error {
	return h.repo.Create(msg)
}

func (h *ChatHistory) GetMessagesByChatID(chatID int) ([]models.ChatMessage, error) {
	return h.repo.Where("chat_id = ?", chatID)
}

func (h *ChatHistory) GetMessagesBySenderID(senderID int) ([]models.ChatMessage, error) {
	return h.repo.Where("sender_id = ?", senderID)
}

func NewChatHistory(repo database.Repository[models.ChatMessage]) *ChatHistory {
	return &ChatHistory{repo: repo}
}
