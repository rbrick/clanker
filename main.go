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
	"github.com/rbrick/clanker/objectstore"
	"github.com/rbrick/clanker/platform"
	"github.com/rbrick/clanker/services"
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

	database.Migrate(db, models.ChatMessage{}, models.AllowlistEntry{}, models.Snippet{}, models.SnippetFile{}, models.SnippetGitFile{}, models.Blob{}, models.ServiceConnection{}, models.ServiceOAuthState{})

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
	publicBaseURL := env.GetEnv("PUBLIC_BASE_URL", env.GetEnv("PUBLIC_WEB_URL", "http://localhost"+env.GetEnv("API_ADDR", ":8080")))
	snippetOptions := []snippets.Option{
		snippets.WithGitStore(snippets.NewGitStore(), database.NewRepository[models.SnippetGitFile](DB), publicBaseURL),
	}
	if bucket := os.Getenv("AWS_S3_BUCKET_NAME"); bucket != "" {
		bucketStore, err := objectstore.NewS3Store(context.Background(), objectstore.S3Config{
			Bucket:          bucket,
			Region:          os.Getenv("AWS_DEFAULT_REGION"),
			Endpoint:        os.Getenv("AWS_ENDPOINT_URL"),
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			Prefix:          os.Getenv("AWS_S3_PREFIX"),
			ForcePathStyle:  env.GetEnv("AWS_S3_FORCE_PATH_STYLE", "true") == "true",
		})
		if err != nil {
			panic(err)
		}
		snippetOptions = []snippets.Option{snippets.WithGitObjectStore(snippets.NewGitStore(), bucketStore, publicBaseURL)}
	}
	snippetStore := snippets.NewSnippets(
		database.NewRepository[models.Snippet](DB),
		database.NewRepository[models.SnippetFile](DB),
		snippetOptions...,
	)
	mediaStore := media.NewStore(database.NewRepository[models.Blob](DB))
	allowlistStore := allowlist.NewAllowlist(database.NewRepository[models.AllowlistEntry](DB))
	serviceManager := services.NewManager(database.NewRepository[models.ServiceConnection](DB), database.NewRepository[models.ServiceOAuthState](DB), publicBaseURL)

	internalTools := []fantasy.AgentTool{}
	internalTools = append(internalTools, tools.NewChatHistoryTool(history, summarizer).Tools()...)
	internalTools = append(internalTools, tools.NewSnippetsTool(snippetStore).Tools()...)
	internalTools = append(internalTools, tools.NewImageGeneratorTool(mediaStore, publicBaseURL).Tools()...)
	internalTools = append(internalTools, tools.NewLinearTool(serviceManager).Tools()...)

	clanker, err := makeAgent(provider, internalTools...)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiServer := api.NewServer(env.GetEnv("API_ADDR", ":8080"), snippetStore, mediaStore, serviceManager)
	go func() {
		log.Printf("starting read-only api on %s", env.GetEnv("API_ADDR", ":8080"))
		if err := apiServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("api stopped: %v", err)
			stop()
		}
	}()

	platforms := []platform.Platform{}
	if botKey := os.Getenv("TELEGRAM_BOT_KEY"); botKey != "" {
		platforms = append(platforms, platform.NewTelegramPlatform(botKey, clanker, allowlistStore, history, serviceManager))
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
