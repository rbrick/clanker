package platform

import (
	"context"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TelegramPlatform struct {
	BotKey     string
	botHandler *bot.Bot
}

func (t *TelegramPlatform) handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Handle incoming messages here

	if update.Message != nil {
		// Process the message and call the registered handler

		// TODO: work on handling audio, video, and other media typess
		// if update.Message.Audio != nil {
		// 	audioUrl, err := b.GetFile(ctx, &bot.GetFileParams{
		// 		FileID: update.Message.Audio.FileID,
		// 	})

		// 	if err != nil {s
		// 		// Handle error

		// 	}
		// }

		t.HandleMessage(&Message{
			Sender: &Sender{
				Platform: "telegram",
				ID:       strconv.Itoa(int(update.Message.Chat.ID)),
				Username: update.Message.Chat.Username,
			},
			Content: &Content{
				Text: update.Message.Text,
			},
			ID: strconv.Itoa(update.Message.ID),
		})

	}
}

func (t *TelegramPlatform) Init() error {
	var err error

	options := []bot.Option{
		bot.WithDefaultHandler(t.handle),
	}

	t.botHandler, err = bot.New(t.BotKey, options...)

	if err != nil {
		return err
	}

	return nil
}

func (t *TelegramPlatform) HandleMessage(msg *Message) {

}

type TelegramBotHandler struct{}

func NewTelegramPlatform(botKey string) *TelegramPlatform {
	return &TelegramPlatform{
		BotKey: botKey,
	}
}
