package bot

import (
	"context"
	"errors"
	"fmt"
	"github-watcher/internal/logger"
	"github-watcher/internal/repository"
	"github-watcher/internal/storage"
	"github-watcher/internal/tracker"
	"github-watcher/models"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot               *tgbotapi.BotAPI
	repositoryService *repository.Service
	tracker           *tracker.Tracker
	storage           storage.Storage
}
type CollectResult struct {
	Repository models.Repository
	err        error
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
		response = helpMessage()
	case "add":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		response, err = h.handleAdd(ctx, fullName)
		defer cancel()
		if err != nil {
			if errors.Is(err, storage.ErrRepositoryAlreadyTracked) {
				response = "Этот репозиторий уже отслеживается."
			} else {
				slog.Error("error handling add command", logger.Err(err))
				response = "Не удалось выполнить команду add. Попробуй позже."
			}
		}
	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		response, err = h.handleList(ctx)
		if err != nil {
			slog.Error("error handling list command", logger.Err(err))
			response = "Не удалось выполнить команду list. Попробуй позже."
		}
	case "start":
		response = startMessage()
	case "collect":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err = h.collectHandler(ctx)
		defer cancel()
		if err != nil {
			slog.Error("error handling collect command", logger.Err(err))
			response = "не удалось выполнить команду collect. Попробуйте позже."
		}
	case "stats":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, err = h.statsHandler(ctx)
		defer cancel()
		if err != nil {
			slog.Error("error handling stats command", logger.Err(err))
			response = "не удалось выполнить команду stats. Попробуйте позже."
		}
	default:
		response = "Неизвестная команда. Используй /help."
	}
	message := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	if _, err := h.bot.Send(message); err != nil {
		slog.Error("error sending message", logger.Err(err))
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
	jobs := make(chan models.Repository)
	results := make(chan CollectResult)

	wg := &sync.WaitGroup{}

	correctCollectCount := 0
	badCollectCount := 0
	workers := 5
	repos, err := h.repositoryService.List(ctx)
	if err != nil {
		return "", fmt.Errorf("не получилось получить список репозиториев %w", err)
	}
	if len(repos) == 0 {
		return "список репозиториев пуст добавьте например flecc1/URL-Shortener", nil
	}

	go func() {
		defer close(jobs)
		for _, repo := range repos {
			select {
			case jobs <- repo:
			case <-ctx.Done():
				return
			}
		}
	}()

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case job, ok := <-jobs:
					if !ok {
						return
					}

					err := h.tracker.TrackRepository(ctx, job)

					collectResult := CollectResult{
						Repository: job,
						err:        err,
					}

					select {
					case results <- collectResult:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			badCollectCount++
		} else {
			correctCollectCount++
		}
	}
	if correctCollectCount == 0 {
		return "Сбор статистики завершился с ошибкой для всех репозиториев.", nil
	}
	if badCollectCount == 0 {
		return fmt.Sprintf("Статистика всех репозиториев собрана успешно: %d.", correctCollectCount), nil
	}
	return fmt.Sprintf("Сбор завершён. Успешно: %d, с ошибками: %d.", correctCollectCount, badCollectCount), nil
}

func (h *Handler) statsHandler(ctx context.Context) (string, error) {
	repos, err := h.repositoryService.List(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка получения данных о репозиториях %w", err)
	}
	if len(repos) == 0 {
		return "Список отслеживаемых репозиториев пуст. Добавь первый: /add flecc1/URL-Shortener", nil
	}

	var response strings.Builder

	for _, repo := range repos {
		reposSnapshots, err := h.storage.ListSnapshots(ctx, repo.ID)
		if err != nil {
			slog.Error("error getting repositories snapshots", "repository", repo.FullName,
				"error", err)
			fmt.Fprintf(&response, "не удалось получить снимки репозитория %s.\n", repo.FullName)
			continue
		}
		if len(reposSnapshots) == 0 {
			fmt.Fprintf(&response, "репозиторий %s не имеет снимков\n", repo.FullName)
		} else {
			reposSnapshot := reposSnapshots[0]
			fmt.Fprintf(&response, "📦 %s\n"+
				"⭐ Stars: %d\n"+
				"🍴 Forks: %d\n"+
				"🐛 Open issues: %d\n"+
				"🔀 Open PRs: %d\n"+
				"💻 Commits за 7 дней: %d\n"+
				"📄 README: %t\n"+
				"🕒 Последний сбор: %s\n"+
				"━━━━━━━━━━━━━━━━━━━━\n", repo.FullName, reposSnapshot.Stars, reposSnapshot.Forks,
				reposSnapshot.OpenIssues, reposSnapshot.OpenPullRequests, reposSnapshot.CommitsLast7Days,
				reposSnapshot.HasReadme, reposSnapshot.CollectedAt)
		}
	}
	return response.String(), nil
}

func startMessage() string {
	return "Привет! 👋 Я GitHub Watcher — бот для отслеживания статистики GitHub-репозиториев.\n" +
		"Что я умею?\n" +
		"- ⭐ Показывать звёзды и форки.\n" +
		"- 🐛 Отслеживать открытые issues и pull requests.\n" +
		"- 💻 Показывать количество коммитов за последние 7 дней.\n" +
		"- 📊 Сохранять историю собранной статистики.\n" +
		"Быстрый старт:\n1." +
		" Добавь репозиторий: /add owner/repo\n2" +
		". Собери статистику: /collect\n3. " +
		"Посмотри результаты: /stats\nИспользуй /help, чтобы узнать обо всех командах."
}

func helpMessage() string {
	return "Доступные команды:\n" +
		"/start — приветствие и быстрый старт\n" +
		"/help — показать список команд\n" +
		"/add owner/repo — добавить репозиторий\n" +
		"/list — показать отслеживаемые репозитории\n" +
		"/collect — собрать и сохранить свежую статистику\n" +
		"/stats — показать последнюю сохранённую статистику"
}
