package platform

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rbrick/clanker/agent"
	"github.com/rbrick/clanker/allowlist"
	"github.com/rbrick/clanker/text"
)

type TelegramPlatform struct {
	BotKey     string
	botHandler *bot.Bot
	Agent      agent.Agent

	Allowlist *allowlist.Allowlist
}

func (*TelegramPlatform) Init() error {
	// botHandler, err := bot.New()
	return nil
}

func (*TelegramPlatform) Start(ctx context.Context) error {
	return nil
}

func (t *TelegramPlatform) handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Handle incoming messages here
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

func (t *TelegramPlatform) mentions(ctx context.Context, msg *text.Message) bool {

	if msg.Chat.Type == "private" {
		return true
	}

	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		botInfo, err := t.botHandler.GetMe(ctx)

		if err != nil {
			return false
		}

		botUsername := botInfo.Username

		if msg.RepliedTo != nil && strings.EqualFold(strconv.Itoa(int(botInfo.ID)), msg.RepliedTo.ID) {
			return true
		}

		if strings.Contains(strings.ToLower(msg.Content.Text), "clanker") || strings.Contains(strings.ToLower(msg.Content.Text), strings.ToLower(botUsername)) {
			return true
		}

	}

	return false
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func (t *TelegramPlatform) HandleMessage(ctx context.Context, msg *text.Message) error {

	if !t.mentions(ctx, msg) {
		log.Printf("Ignoring message: %v", msg)
		return nil
	}

	if allowed, err := t.Allowlist.IsAllowed("telegram", msg.Chat.ID); err != nil {
		log.Printf("Error checking allowlist: %v", err)
		return nil
	} else if !allowed {

		chatId := mustAtoi(msg.Chat.ID)
		log.Printf("Chat %d is not allowed to interact with the bot", chatId)

		t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			ReplyParameters: &models.ReplyParameters{
				MessageID: mustAtoi(msg.ID),
			},
			Text: "❌ This chat is not allowed to interact with the bot.",
		})

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
		ParseMode: "Markdown",
	})

	if err != nil {
		return err
	}
	log.Printf("Sent message: %v", sentMsg)
	return nil
}

func NewTelegramPlatform(botKey string, a agent.Agent) *TelegramPlatform {
	return &TelegramPlatform{
		BotKey: botKey,
		Agent:  a,
	}
}
