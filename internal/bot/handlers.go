package bot

import (
	"context"
	"errors"
	"fmt"
	"github-watcher/internal/repository"
	"github-watcher/internal/storage"
	"github-watcher/internal/tracker"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot               *tgbotapi.BotAPI
	repositoryService *repository.Service
	tracker           *tracker.Tracker
}

func NewHandler(bot *tgbotapi.BotAPI, repositoryService *repository.Service, tracker tracker.Tracker) *Handler {
	return &Handler{
		bot:               bot,
		repositoryService: repositoryService,
		tracker:           &tracker,
	}
}

func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	var response string
	var err error

	fullName := update.Message.CommandArguments()
	switch update.Message.Command() {
	case "help":
		response = "Доступные команды:\n" +
			"/add owner/repo — добавить репозиторий\n" +
			"/list — показать репозитории"
	case "add":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		response, err = h.handleAdd(ctx, fullName)
		cancel()
		if err != nil {
			if errors.Is(err, storage.ErrRepositoryAlreadyTracked) {
				response = "Этот репозиторий уже отслеживается."
			} else {
				slog.Error("error handling add command", "error", err)
				response = "Не удалось выполнить команду add. Попробуй позже."
			}
		}
	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, err = h.handleList(ctx)
		cancel()
		if err != nil {
			slog.Error("error handling list command", "error", err)
			response = "Не удалось выполнить команду list. Попробуй позже."
		}
	case "start":
		response = "Привет! Я GitHub Watcher.\n\n" +
			"/add owner/repo — начать отслеживать репозиторий\n" +
			"/list — показать отслеживаемые репозитории"
	default:
		response = "Неизвестная команда. Используй /help."
	}
	message := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	if _, err := h.bot.Send(message); err != nil {
		slog.Error("error sending message", "error", err)
	}
}

func (h *Handler) handleAdd(ctx context.Context, fullName string) (string, error) {
	if strings.TrimSpace(fullName) == "" {
		return "Укажи репозиторий в формате owner/repo.\nПример: /add golang/go", nil
	}
	repo, err := h.repositoryService.Add(ctx, fullName)
	if err != nil {
		return "", fmt.Errorf("failed to add repository: %w", err)
	}
	return fmt.Sprintf("Репозиторий %s успешно добавлен.", repo.FullName), nil
}

func (h *Handler) handleList(ctx context.Context) (string, error) {
	repos, err := h.repositoryService.List(ctx)
	if err != nil {
		return "", fmt.Errorf("не удалось получить список репозиториев: %w", err)
	}

	if len(repos) == 0 {
		return "Пока нет отслеживаемых репозиториев. Добавь первый: /add golang/go", nil
	}

	var response strings.Builder
	response.WriteString("Отслеживаемые репозитории:\n")

	for i, repo := range repos {
		fmt.Fprintf(&response, "%d. %s\n", i+1, repo.FullName)
	}
	return response.String(), nil
}
