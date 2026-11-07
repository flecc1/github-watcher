package main

import (
	"context"
	"errors"
	"github-watcher/internal/bot"
	"github-watcher/internal/logger"
	"github-watcher/internal/scheduler"
	"github-watcher/internal/service"
	"github-watcher/internal/tracker"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github-watcher/internal/github"
	"github-watcher/internal/postgres"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

const longPolling = 60
const defaultCollectInterval = 6 * time.Hour

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

	collectInterval := defaultCollectInterval
	if value := os.Getenv("COLLECT_INTERVAL"); value == "" {
		slog.Info("COLLECT_INTERVAL not set, using default", "interval", defaultCollectInterval)
	} else if parsed, err := time.ParseDuration(value); err != nil || parsed <= 0 {
		slog.Warn("Invalid COLLECT_INTERVAL, using default", "value", value, "interval", defaultCollectInterval)
	} else {
		collectInterval = parsed
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

	githubToken := os.Getenv("GITHUB_TOKEN")

	store := postgres.NewStore(db)
	githubClient := github.NewClient(githubToken)
	snapshotsTracker := tracker.NewTracker(githubClient, store)
	repositoryService := service.NewService(githubClient, store, snapshotsTracker)

	botAPI, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, &http.Client{
		Timeout: longPolling*time.Second + 30*time.Second,
	})
	if err != nil {
		slog.Error("Failed to create bot api", logger.Err(err))
		return
	}

	handler := bot.NewHandler(botAPI, repositoryService)
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
	} else {
		slog.Info("Bot commands registered")
	}

	sch := scheduler.NewScheduler(repositoryService, collectInterval)

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workCtx, cancelWork := context.WithCancel(context.Background())
	defer cancelWork()

	schDone := make(chan struct{})
	go func() {
		defer close(schDone)
		sch.Run(signalCtx)
	}()

	u := tgbotapi.NewUpdate(0)

	u.Timeout = longPolling
	updates := botAPI.GetUpdatesChan(u)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-signalCtx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				handler.HandleUpdate(workCtx, update)
			}
		}
	}()

	allDone := make(chan struct{})
	go func() {
		<-done
		<-schDone
		close(allDone)
	}()

	<-signalCtx.Done()
	stop()
	slog.Info("Shutting down gracefully")

	botAPI.StopReceivingUpdates()

	select {
	case <-allDone:
	case <-time.After(20 * time.Second):
		slog.Warn("Shutdown timeout reached, cancelling active command")
		cancelWork()

		select {
		case <-allDone:
		case <-time.After(3 * time.Second):
			slog.Warn("Active command did not stop in time")
		}
	}

	slog.Info("Stop bot")
}
