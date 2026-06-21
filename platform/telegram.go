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
	"github.com/rbrick/clanker/chat"
	dbmodels "github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/services"
	"github.com/rbrick/clanker/text"
)

type TelegramPlatform struct {
	BotKey     string
	botHandler *bot.Bot
	Agent      agent.Agent

	Allowlist      *allowlist.Allowlist
	ChatHistory    *chat.ChatHistory
	ServiceManager *services.Manager
}

func (t *TelegramPlatform) Init() error {
	botHandler, err := bot.New(t.BotKey, bot.WithDefaultHandler(t.handle), bot.WithCallbackQueryDataHandler("connect:", bot.MatchTypePrefix, t.handleConnectCallback))
	if err != nil {
		return err
	}
	t.botHandler = botHandler
	return nil
}

func (t *TelegramPlatform) Start(ctx context.Context) error {
	t.botHandler.Start(ctx)
	return ctx.Err()
}

func (t *TelegramPlatform) Config() *PlatformConfig {
	return &PlatformConfig{
		Instructions: "You are responding on Telegram. Keep replies concise and compatible with Telegram Markdown.",
	}
}

func (t *TelegramPlatform) handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Handle incoming messages here
	if update.Message == nil {
		return
	}

	if update.Message != nil {
		if update.Message.From == nil {
			return
		}

		messageText := update.Message.Text
		if messageText == "" {
			messageText = update.Message.Caption
		}

		msg := &text.Message{
			Platform: "telegram",
			Sender: &text.Chatter{
				ID:       strconv.Itoa(int(update.Message.From.ID)),
				Username: update.Message.From.Username,
				Name:     strings.TrimSpace(update.Message.From.FirstName + " " + update.Message.From.LastName),
			},
			Content: &text.Content{
				Text: messageText,
			},
			Chat: &text.Chat{
				ID:   strconv.Itoa(int(update.Message.Chat.ID)),
				Type: string(update.Message.Chat.Type),
			},
			ID:        strconv.Itoa(update.Message.ID),
			Timestamp: int64(update.Message.Date),
		}

		if update.Message.ReplyToMessage != nil {
			msg.ReplyToMessageID = strconv.Itoa(update.Message.ReplyToMessage.ID)
			if update.Message.ReplyToMessage.From != nil {
				msg.RepliedTo = &text.Chatter{
					ID:       strconv.Itoa(int(update.Message.ReplyToMessage.From.ID)),
					Username: update.Message.ReplyToMessage.From.Username,
					Name:     strings.TrimSpace(update.Message.ReplyToMessage.From.FirstName + " " + update.Message.ReplyToMessage.From.LastName),
				}
			}
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic handling Telegram message %s in chat %s: %v", msg.ID, msg.Chat.ID, r)
				}
			}()
			if err := t.HandleMessage(ctx, msg); err != nil {
				log.Printf("Error handling Telegram message %s in chat %s: %v", msg.ID, msg.Chat.ID, err)
			}
		}()
	}
}

func (t *TelegramPlatform) mentions(ctx context.Context, msg *text.Message) bool {
	if msg == nil || msg.Chat == nil || msg.Content == nil {
		return false
	}

	if msg.Chat.Type == "private" {
		return true
	}

	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		botInfo, err := t.botHandler.GetMe(ctx)

		if err != nil {
			log.Printf("Unable to determine bot username for mention detection; ignoring group message %s in chat %s: %v", msg.ID, msg.Chat.ID, err)
			return false
		}

		botUsername := botInfo.Username

		if msg.RepliedTo != nil && strings.EqualFold(strconv.Itoa(int(botInfo.ID)), msg.RepliedTo.ID) {
			return true
		}

		text := strings.ToLower(msg.Content.Text)
		hasBotName := strings.Contains(text, "clanker") || strings.Contains(text, strings.ToLower(botUsername))
		return hasBotName

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

	if err := t.saveMessage(msg); err != nil {
		log.Printf("Error saving chat message: %v", err)
	}

	if strings.TrimSpace(msg.Content.Text) == "/connect" || strings.HasPrefix(strings.TrimSpace(msg.Content.Text), "/connect@") {
		return t.sendConnectMenu(ctx, msg)
	}

	if !t.mentions(ctx, msg) {
		chatID := ""
		msgID := ""
		text := ""
		if msg.Chat != nil {
			chatID = msg.Chat.ID
		}
		if msg.Content != nil {
			text = msg.Content.Text
		}
		msgID = msg.ID
		log.Printf("Ignoring Telegram message %s in chat %s; not addressed to bot. text=%q", msgID, chatID, text)
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

	if err := t.attachContext(msg, 40); err != nil {
		log.Printf("Error loading chat context: %v", err)
	}

	reply, err := t.Agent.Generate(ctx, *msg)

	if err != nil {
		log.Printf("Agent failed to generate reply: %v", err)
		messageID, _ := strconv.Atoi(msg.ID)
		_, sendErr := t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Sorry, I hit an internal error while generating a response. Try again or check the logs.",
			ReplyParameters: &models.ReplyParameters{
				MessageID: messageID,
			},
		})
		if sendErr != nil {
			log.Printf("Failed to send agent error message: %v", sendErr)
		}
		return err
	}

	log.Println(reply)

	messageID, _ := strconv.Atoi(msg.ID)

	var sentMsg *models.Message
	if reply.Content.ImageURL != "" {
		caption := withMediaLink(reply.Content.Text, reply.Content.ImageURL)
		sentMsg, err = t.botHandler.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:  msg.Chat.ID,
			Photo:   &models.InputFileString{Data: reply.Content.ImageURL},
			Caption: caption,
			ReplyParameters: &models.ReplyParameters{
				MessageID: messageID,
			},
			ParseMode: "Markdown",
		})
	} else {
		sentMsg, err = t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   reply.Content.Text,
			ReplyParameters: &models.ReplyParameters{
				MessageID: messageID,
			},
			ParseMode: "Markdown",
		})
	}

	if err != nil {
		log.Printf("Failed to send Telegram reply with Markdown, retrying as plain text: %v", err)
		fallbackText := reply.Content.Text
		if reply.Content.ImageURL != "" {
			fallbackText = withMediaLink(reply.Content.Text, reply.Content.ImageURL)
		}
		sentMsg, err = t.botHandler.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fallbackText,
			ReplyParameters: &models.ReplyParameters{
				MessageID: messageID,
			},
		})
		if err != nil {
			return err
		}
	}
	log.Printf("Sent message: %v", sentMsg)

	if sentMsg.From != nil {
		if err := t.saveMessage(&text.Message{
			ID:               strconv.Itoa(sentMsg.ID),
			Timestamp:        int64(sentMsg.Date),
			Platform:         msg.Platform,
			ReplyToMessageID: msg.ID,
			Chat:             msg.Chat,
			Sender: &text.Chatter{
				ID:       strconv.Itoa(int(sentMsg.From.ID)),
				Username: sentMsg.From.Username,
				Name:     strings.TrimSpace(sentMsg.From.FirstName + " " + sentMsg.From.LastName),
			},
			Content: reply.Content,
		}); err != nil {
			log.Printf("Error saving bot chat message: %v", err)
		}
	}

	return nil
}

func withMediaLink(textValue, url string) string {
	if url == "" || strings.Contains(textValue, url) {
		return textValue
	}
	mediaLine := "Clanker media link: " + url
	if strings.TrimSpace(textValue) == "" {
		return mediaLine
	}
	return strings.TrimSpace(textValue) + "\n\n" + mediaLine
}

func (t *TelegramPlatform) attachContext(msg *text.Message, limit int) error {
	if t.ChatHistory == nil || msg == nil || msg.Chat == nil {
		return nil
	}

	messages, err := t.ChatHistory.GetRecentMessages(msg.Platform, mustAtoi(msg.Chat.ID), limit)
	if err != nil {
		return err
	}

	msg.Context = make([]text.ContextMessage, 0, len(messages))
	for _, m := range messages {
		messageID := m.MessageID
		if messageID == "" {
			messageID = strconv.Itoa(m.ID)
		}

		ctxMsg := text.ContextMessage{
			ID:        messageID,
			Timestamp: m.Timestamp,
			Sender: &text.Chatter{
				ID:       strconv.Itoa(m.SenderID),
				Username: m.SenderUsername,
				Name:     m.SenderName,
			},
			Content: &text.Content{Text: m.Message},
		}
		if m.ReplyToID != nil {
			ctxMsg.ReplyToMessageID = strconv.Itoa(*m.ReplyToID)
		}
		msg.Context = append(msg.Context, ctxMsg)
	}

	return nil
}

func (t *TelegramPlatform) saveMessage(msg *text.Message) error {
	if t.ChatHistory == nil || msg == nil || msg.Chat == nil || msg.Sender == nil || msg.Content == nil {
		return nil
	}

	var replyToID *int
	if msg.ReplyToMessageID != "" {
		id := mustAtoi(msg.ReplyToMessageID)
		replyToID = &id
	}

	return t.ChatHistory.SaveMessage(&dbmodels.ChatMessage{
		MessageID:      msg.ID,
		Platform:       msg.Platform,
		ChatID:         mustAtoi(msg.Chat.ID),
		ReplyToID:      replyToID,
		SenderID:       mustAtoi(msg.Sender.ID),
		SenderUsername: msg.Sender.Username,
		SenderName:     msg.Sender.Name,
		Message:        msg.Content.Text,
		Timestamp:      msg.Timestamp,
	})
}

func NewTelegramPlatform(botKey string, a agent.Agent, allowlist *allowlist.Allowlist, history *chat.ChatHistory, serviceManager *services.Manager) *TelegramPlatform {
	return &TelegramPlatform{
		BotKey:         botKey,
		Agent:          a,
		Allowlist:      allowlist,
		ChatHistory:    history,
		ServiceManager: serviceManager,
	}
}
