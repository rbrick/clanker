package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"github.com/joho/godotenv"
	"github.com/rbrick/clanker/agent"
	"github.com/rbrick/clanker/allowlist"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/env"
	"github.com/rbrick/clanker/platform"
	"github.com/rbrick/clanker/snippets"
	"github.com/rbrick/clanker/tools"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	godotenv.Load()

	db, err := initializeDatabase()
	if err != nil {
		panic(err)
	}

	DB = db
}

func makeAgent(agentTools ...fantasy.AgentTool) (agent.Agent, error) {
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

	agent, err := agent.NewClanker(context.Background(), os.Getenv("LLM_MODEL"), provider, agentTools...)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func initializeDatabase() (*gorm.DB, error) {

	dbUri := env.GetEnv("DB_URI", "clanker.db")
	dbProvider := env.GetEnv("DB_PROVIDER", "sqlite")

	db, err := database.Open(dbProvider, dbUri)

	if err != nil {
		return nil, err
	}

	database.Migrate(db, models.ChatMessage{}, models.AllowlistEntry{})

	return db, nil
}

func main() {

	snippets := snippets.NewSnippets(database.NewRepository[models.Snippet](DB))

	internalTools := []fantasy.AgentTool{}

	internalTools = append(internalTools, tools.NewSnippetsTool(snippets).Tools()...)

	agent, err := makeAgent(internalTools...)

	if err != nil {
		panic(err)
	}

	telegramPlatform := &platform.TelegramPlatform{
		BotKey:    os.Getenv("PLATFORM_TELEGRAM_BOT_TOKEN"),
		Agent:     agent,
		Allowlist: allowlist.NewAllowlist(database.NewRepository[models.AllowlistEntry](DB)),
	}

	if err := telegramPlatform.Init(); err != nil {
		panic(err)
	}

	if err := telegramPlatform.Start(context.Background()); err != nil {
		panic(err)
	}

	select {}
}
