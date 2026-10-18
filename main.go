package main

import (
	"context"
	"errors"
	"github-watcher/internal/bot"
	"github-watcher/internal/logger"
	"github-watcher/internal/service"
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
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Could not load .env file", logger.Err(err))
		}
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
		slog.Error("Failed to open database", logger.Err(err))
		return
	}
	defer db.Close()

	slog.Info("Connected to database")

	store := postgres.NewStore(db)
	githubClient := github.NewClient()

	repositoryService := service.NewService(githubClient, store)
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		slog.Error("Failed to create bot api", logger.Err(err))
		return
	}
	snapshotsTracker := tracker.NewTracker(githubClient, store)
	handler := bot.NewHandler(botAPI, repositoryService, snapshotsTracker, store)
	slog.Info("Bot authorized", "username", botAPI.Self.UserName)

	commands := []tgbotapi.BotCommand{
		{
			Command:     "start",
			Description: "Запустить бота",
		},
		{
			Command:     "help",
			Description: "Показать справку по командам",
		},
		{
			Command:     "add",
			Description: "Добавить репозиторий в формате owner/repo",
		},
		{
			Command:     "list",
			Description: "Показать отслеживаемые репозитории",
		},
		{
			Command:     "collect",
			Description: "Собрать свежую статистику репозиториев",
		},
		{
			Command:     "stats",
			Description: "Показать сохраненную статистику",
		},
		{
			Command:     "delete",
			Description: "Удалить репозиторий в формате owner/repo",
		},
	}

	config := tgbotapi.NewSetMyCommands(commands...)
	_, err = botAPI.Request(config)
	if err != nil {
		slog.Error("Failed to register bot commands", logger.Err(err))
	}
	slog.Info("Bot commands registered")

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	for update := range updates {
		handler.HandleUpdate(update)
	}
}
