package platform

import (
	"context"
	"log"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rbrick/clanker/services"
	"github.com/rbrick/clanker/text"
)

func (t *TelegramPlatform) sendConnectMenu(ctx context.Context, msg *text.Message) error {
	if t.ServiceManager == nil {
		_, err := t.botHandler.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "No connectable services are configured."})
		return err
	}
	_, err := t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "Choose a service to connect for this chat:",
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: "Linear", CallbackData: "connect:" + services.LinearService}}}},
	})
	return err
}

func (t *TelegramPlatform) handleConnectCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil || t.ServiceManager == nil {
		return
	}
	chatID := int(update.CallbackQuery.Message.Message.Chat.ID)
	service := update.CallbackQuery.Data[len("connect:"):]
	url, err := t.ServiceManager.BeginOAuth("telegram", chatID, service)
	text := "Open this link to connect " + service + ":\n" + url
	if err != nil {
		text = "Could not start connection: " + err.Error()
	}
	_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: strconv.Itoa(chatID), Text: text})
	if sendErr != nil {
		log.Printf("failed to send connect link: %v", sendErr)
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
}
