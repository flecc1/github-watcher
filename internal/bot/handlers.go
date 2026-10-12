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
	storage           storage.Storage
}

func NewHandler(bot *tgbotapi.BotAPI, repositoryService *repository.Service, tracker *tracker.Tracker, storage storage.Storage) *Handler {
	return &Handler{
		bot:               bot,
		repositoryService: repositoryService,
		tracker:           tracker,
		storage:           storage,
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
			"/list — показать репозитории\n" +
			"/collect - собрать статистику о снимках" +
			"/stats - вывести статистику"
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
			"/list — показать отслеживаемые репозитории\n" +
			"/collect - собрать статистику о снимках\n" +
			"/stats - вывести статистику"
	case "collect":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err = h.collectHandler(ctx)
		cancel()
		if err != nil {
			slog.Error("error handling collect command", "error", err)
			response = "не удалось выполнить команду collect. Попробуйте позже."
		}
	case "stats":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, err = h.statsHandler(ctx)
		cancel()
		if err != nil {
			slog.Error("error handling stats command", "error", err)
			response = "не удалось выполнить команду stats. Попробуйте позже."
		}
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

func (h *Handler) collectHandler(ctx context.Context) (string, error) {
	correctCollectCount := 0
	badCollectCount := 0
	repos, err := h.repositoryService.List(ctx)
	if err != nil {
		return "", fmt.Errorf("не получилось получить список репозиториев %w", err)
	}
	if len(repos) == 0 {
		return "список репозиториев пуст добавьте например flecc1/URL-Shortener", nil
	}
	for _, repo := range repos {
		err := h.tracker.TrackRepository(ctx, repo)
		if err != nil {
			slog.Error("error tracking repository", "repository", repo.FullName, "error", err)
			badCollectCount++
		} else {
			correctCollectCount++
		}
	}
	return fmt.Sprintf("успешно собрана %d а %d получились с ошибкой",
		correctCollectCount, badCollectCount), nil
}

func (h *Handler) statsHandler(ctx context.Context) (string, error) {
	repos, err := h.repositoryService.List(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка получения данных о репозиториях %w", err)
	}
	if len(repos) == 0 {
		return "Список отслеживаемых репозиториев пуст. Добавь первый: /add flecc1-URL-Shortener", nil
	}

	var response strings.Builder

	for _, repo := range repos {
		reposSnapshots, err := h.storage.ListSnapshots(ctx, repo.ID)
		if err != nil {
			slog.Error("error getting repositories snapshots", "repository", repo.FullName,
				"error", err)
			fmt.Fprintf(&response, "ошибка при получения снимков репозитория %s.\n", repo.FullName)
			continue
		}
		if len(reposSnapshots) == 0 {
			fmt.Fprintf(&response, "репозиторий %s не имеет снимков\n", repo.FullName)
		} else {
			reposSnapshot := reposSnapshots[0]
			fmt.Fprintf(&response, "репозиторий %s имеет: Stars %d,"+
				" Forks %d, OpenIssues %d, OpenPullRequests %d, CommitsLast7Days %d,"+
				" README %t\n", repo.FullName, reposSnapshot.Stars, reposSnapshot.Forks,
				reposSnapshot.OpenIssues, reposSnapshot.OpenPullRequests, reposSnapshot.CommitsLast7Days,
				reposSnapshot.HasReadme)
		}
	}
	return response.String(), nil
}
