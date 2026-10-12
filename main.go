package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github-watcher/internal/github"
	"github-watcher/internal/postgres"
	"github-watcher/internal/tracker"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("Could not load .env file", "error", err)
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := postgres.Open(dbCtx, dsn)
	dbCancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	slog.Info("Connected to database")

	store := postgres.NewStore(db)
	githubClient := github.NewClient()
	repoTracker := tracker.NewTracker(githubClient, store)

	listCtx, listCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	repositories, err := store.ListRepositories(listCtx)
	listCancel()

	if err != nil {
		log.Fatal(err)
	}

	for _, repo := range repositories {
		trackCtx, trackCancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

		err := repoTracker.TrackRepository(trackCtx, repo)
		trackCancel()

		if err != nil {
			slog.Error(
				"Failed to track repository",
				"repository", repo.FullName,
				"error", err,
			)
			continue
		}

		slog.Info(
			"Repository snapshot saved",
			"repository", repo.FullName,
		)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Bot authorized", "username", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		var response string

		switch update.Message.Command() {
		case "start":
			response = "Привет! Я GitHub Watcher.\n\n" +
				"Я помогу следить за статистикой GitHub-репозиториев.\n\n" +
				"Доступные команды:\n" +
				"/start — запуск бота\n" +
				"/help — список команд"
		case "help":
			response = "Доступные команды:\n" +
				"/start — запуск бота\n" +
				"/help — список команд"
		default:
			response = "Неизвестная команда. Используй /help."
		}

		message := tgbotapi.NewMessage(update.Message.Chat.ID, response)

		if _, err := bot.Send(message); err != nil {
			slog.Error("Failed to send message", "error", err)
		}
	}
}
