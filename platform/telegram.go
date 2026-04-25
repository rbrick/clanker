package platform

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rbrick/clanker/agent"
	"github.com/rbrick/clanker/text"
)

type TelegramPlatform struct {
	BotKey     string
	botHandler *bot.Bot
	Agent      agent.Agent
}

func (t *TelegramPlatform) handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Handle incoming messages here

	log.Println(update.Message)

	if update.Message == nil {
		return
	}

	if update.Message != nil {
		msg := &text.Message{
			Platform: "telegram",
			Sender: &text.Chatter{
				ID:       strconv.Itoa(int(update.Message.Chat.ID)),
				Username: update.Message.Chat.Username,
			},
			Content: &text.Content{
				Text: update.Message.Text,
			},
			Chat: &text.Chat{
				ID:   strconv.Itoa(int(update.Message.Chat.ID)),
				Type: string(update.Message.Chat.Type),
			},
			ID: strconv.Itoa(update.Message.ID),
		}

		if update.Message.ReplyToMessage != nil {
			msg.RepliedTo = &text.Chatter{
				ID:       strconv.Itoa(int(update.Message.ReplyToMessage.From.ID)),
				Username: update.Message.ReplyToMessage.From.Username,
			}
		}

		t.HandleMessage(ctx, msg)
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

func (t *TelegramPlatform) Start(ctx context.Context) error {
	t.botHandler.Start(ctx)
	return nil
}

func (t *TelegramPlatform) HandleMessage(ctx context.Context, msg *text.Message) error {

	shouldRespond := false
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		botInfo, err := t.botHandler.GetMe(ctx)

		if err != nil {
			log.Printf("Error getting bot info: %v", err)
			return err
		}

		botUsername := botInfo.Username

		if msg.RepliedTo != nil {
			if msg.RepliedTo.Username == botUsername {
				shouldRespond = true
			}
		}

		if strings.Contains(strings.ToLower(msg.Content.Text), "clanker") || strings.Contains(strings.ToLower(msg.Content.Text), strings.ToLower(botUsername)) {
			shouldRespond = true
		}

	}

	if !shouldRespond {
		log.Printf("Ignoring message: %v", msg)
		return nil
	}

	reply, err := t.Agent.Generate(ctx, *msg)

	if err != nil {
		return err
	}

	log.Println(reply)

	messageID, _ := strconv.Atoi(msg.ID)

	sentMsg, err := t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   reply.Content.Text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: messageID,
		},

		ReplyMarkup: &models.ReplyMarkup{},
		ParseMode:   "Markdown",
	})

	if err != nil {
		return err
	}

	log.Printf("Sent message: %v", sentMsg)
	return nil
}

func (t *TelegramPlatform) Reply(ctx context.Context, msg *text.Message, reply string) error {
	return nil
}

func NewTelegramPlatform(botKey string, a agent.Agent) *TelegramPlatform {
	return &TelegramPlatform{
		BotKey: botKey,
		Agent:  a,
	}
}
