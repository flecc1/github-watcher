package main

import (
	"context"
	"errors"
	"github-watcher/internal/github"
	"github-watcher/internal/postgres"
	"github-watcher/internal/storage"
	"log"
	"log/slog"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	slog.Info("Connected to database")
	store := postgres.NewStore(db)

	githubClient := github.NewClient()

	repoInfo, err := githubClient.GetRepository(ctx, "golang", "go")
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Fetched github repo info",
		"fullName", repoInfo.FullName,
		"stars", repoInfo.Stars,
		"forks", repoInfo.Forks,
		"openIssues", repoInfo.OpenIssues,
	)
	repo, err := store.AddRepository(ctx, "golang/go")
	if err != nil {
		if errors.Is(err, storage.ErrRepositoryAlreadyTracked) {
			slog.Info("Repository already tracked")
		} else {
			log.Fatal(err)
		}
	} else {
		slog.Info("Added repository", "fullName", repo.FullName, "id", repo.ID)
	}

	repositories, err := store.ListRepositories(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, repo := range repositories {
		slog.Info("Tracked repository", "fullName", repo.FullName)
	}

	err = store.DeleteRepository(ctx, "golang/go")
	if err != nil {
		if errors.Is(err, storage.ErrRepositoryNotFound) {
			slog.Info("Repository not found")
		} else {
			log.Fatal(err)
		}
	}
	slog.Info("Deleted repository", "fullName", repo.FullName)

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	//bot.Debug = true

	slog.Info("Bot authorized", "username", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if !update.Message.IsCommand() {
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
