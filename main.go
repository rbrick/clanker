package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"github.com/joho/godotenv"
	"github.com/rbrick/clanker/agent"
	"github.com/rbrick/clanker/platform"
)

func init() {
	godotenv.Load()
}

func makeAgent() (agent.Agent, error) {
	var provider fantasy.Provider
	var err error

	switch llmProvider := os.Getenv("LLM_PROVIDER"); llmProvider {
	case "openai":
		provider, err = openai.New(openai.WithAPIKey(os.Getenv("LLM_API_KEY")))
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llmProvider)
	}

	if err != nil {
		return nil, err
	}

	agent, err := agent.NewClanker(context.Background(), os.Getenv("LLM_MODEL"), provider)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func main() {
	agent, err := makeAgent()
	if err != nil {
		panic(err)
	}

	telegramPlatform := &platform.TelegramPlatform{
		BotKey: os.Getenv("PLATFORM_TELEGRAM_BOT_TOKEN"),
		Agent:  agent,
	}

	if err := telegramPlatform.Init(); err != nil {
		panic(err)
	}

	if err := telegramPlatform.Start(context.Background()); err != nil {
		panic(err)
	}

	select {}
}
