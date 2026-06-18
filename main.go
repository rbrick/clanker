package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openrouter"
	"github.com/joho/godotenv"
	"github.com/rbrick/clanker/agent"
	"github.com/rbrick/clanker/allowlist"
	"github.com/rbrick/clanker/api"
	"github.com/rbrick/clanker/chat"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/env"
	"github.com/rbrick/clanker/media"
	"github.com/rbrick/clanker/platform"
	"github.com/rbrick/clanker/snippets"
	"github.com/rbrick/clanker/summarize"
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

func makeProvider() (fantasy.Provider, error) {
	var provider fantasy.Provider
	var err error

	switch llmProvider := os.Getenv("LLM_PROVIDER"); llmProvider {
	case "openai":
		provider, err = openai.New(openai.WithAPIKey(os.Getenv("LLM_API_KEY")))
	case "openrouter":
		provider, err = openrouter.New(
			openrouter.WithAPIKey(os.Getenv("LLM_API_KEY")),
		)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llmProvider)
	}

	if err != nil {
		return nil, err
	}

	return provider, nil
}

func makeAgent(provider fantasy.Provider, agentTools ...fantasy.AgentTool) (agent.Agent, error) {
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

	database.Migrate(db, models.ChatMessage{}, models.AllowlistEntry{}, models.Snippet{}, models.SnippetFile{}, models.Blob{})

	return db, nil
}

func main() {
	provider, err := makeProvider()
	if err != nil {
		panic(err)
	}

	llm, err := provider.LanguageModel(context.Background(), os.Getenv("LLM_MODEL"))
	if err != nil {
		panic(err)
	}

	history := chat.NewChatHistory(database.NewRepository[models.ChatMessage](DB))
	summarizer := summarize.NewService(llm)
	snippetStore := snippets.NewSnippets(database.NewRepository[models.Snippet](DB), database.NewRepository[models.SnippetFile](DB))
	mediaStore := media.NewStore(database.NewRepository[models.Blob](DB))
	allowlistStore := allowlist.NewAllowlist(database.NewRepository[models.AllowlistEntry](DB))

	internalTools := []fantasy.AgentTool{}
	internalTools = append(internalTools, tools.NewChatHistoryTool(history, summarizer).Tools()...)
	internalTools = append(internalTools, tools.NewSnippetsTool(snippetStore).Tools()...)
	internalTools = append(internalTools, tools.NewImageGeneratorTool(mediaStore, env.GetEnv("PUBLIC_BASE_URL", "http://localhost"+env.GetEnv("API_ADDR", ":8080"))).Tools()...)

	clanker, err := makeAgent(provider, internalTools...)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiServer := api.NewServer(env.GetEnv("API_ADDR", ":8080"), snippetStore, mediaStore)
	go func() {
		log.Printf("starting read-only api on %s", env.GetEnv("API_ADDR", ":8080"))
		if err := apiServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("api stopped: %v", err)
			stop()
		}
	}()

	platforms := []platform.Platform{}
	if botKey := os.Getenv("TELEGRAM_BOT_KEY"); botKey != "" {
		platforms = append(platforms, platform.NewTelegramPlatform(botKey, clanker, allowlistStore, history))
	}

	if len(platforms) == 0 {
		log.Println("no bot platforms configured; API only")
		<-ctx.Done()
		return
	}

	if err := platform.NewManager(platforms...).Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}
