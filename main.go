package main

import (
	"context"
	"github-watcher/internal/bot"
	"github-watcher/internal/repository"
	"github-watcher/internal/tracker"
	"log/slog"
	"os"
	"time"

	"github-watcher/internal/github"
	"github-watcher/internal/postgres"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("Could not load .env file", "error", err)
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		slog.Error("TELEGRAM_BOT_TOKEN environment variable not set")
		return
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL environment variable not set")
		return
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := postgres.Open(dbCtx, dsn)
	dbCancel()
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer db.Close()

	slog.Info("Connected to database")

	store := postgres.NewStore(db)
	githubClient := github.NewClient()

	repositoryService := repository.NewService(githubClient, store)
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		slog.Error("Failed to create bot api", "error", err)
		return
	}
	snapshotsTracker := tracker.NewTracker(githubClient, store)
	handler := bot.NewHandler(botAPI, repositoryService, snapshotsTracker, store)
	slog.Info("Bot authorized", "username", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	for update := range updates {
		handler.HandleUpdate(update)
	}
}
