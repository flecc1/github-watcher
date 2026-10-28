package bot

import (
	"context"
	"errors"
	"fmt"
	"github-watcher/internal/logger"
	"github-watcher/internal/service"
	"github-watcher/internal/storage"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	service *service.Service
}

func NewHandler(bot *tgbotapi.BotAPI, repositoryService *service.Service) *Handler {
	return &Handler{
		bot:     bot,
		service: repositoryService,
	}
}

func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	if !update.Message.IsCommand() {
		message := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"Я понимаю только команды. Используй /help.",
		)

		if _, err := h.bot.Send(message); err != nil {
			slog.Error("error sending message", logger.Err(err))
		}

		return
	}

	var err error
	var response string
	responseSent := false

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
				response = fmt.Sprintf("Не удалось выполнить команду add. Попробуйте позже Error: %s", err)
			}
		}
	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		response, err = h.handleList(ctx)
		if err != nil {
			slog.Error("error handling list command", logger.Err(err))
			response = fmt.Sprintf("Не удалось выполнить команду list. Попробуй позже. Error: %s", err)
		}
	case "start":
		response = startMessage()
	case "collect":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		text := "⏳ Собираю статистику репозиториев. Это может занять некоторое время…"
		message := tgbotapi.NewMessage(update.Message.Chat.ID, text)

		messageSent := false
		timeMessage, err := h.bot.Send(message)
		if err != nil {
			slog.Error("error sending message", logger.Err(err))
		} else {
			messageSent = true
		}

		response, err = h.collectHandler(ctx)
		if err != nil {
			slog.Error("error handling collect command", logger.Err(err))
			response = fmt.Sprintf("не удалось выполнить команду collect. Попробуйте позже. Error: %s", err)
		}

		if messageSent {
			newMessage := tgbotapi.NewEditMessageText(timeMessage.Chat.ID, timeMessage.MessageID, response)
			_, err = h.bot.Send(newMessage)
			if err != nil {
				slog.Error("error sending message", logger.Err(err))
			} else {
				responseSent = true
			}
		}
	case "stats":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, err = h.statsHandler(ctx)
		defer cancel()
		if err != nil {
			slog.Error("error handling stats command", logger.Err(err))
			response = fmt.Sprintf("не удалось выполнить команду stats. Попробуйте позже. Error: %s", err)
		}
	case "delete":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		response, err = h.handleDelete(ctx, fullName)
		if err != nil {
			slog.Error("error handling delete command", logger.Err(err))
			response = fmt.Sprintf("не удалось выполнить команду delete. Попробуйте позже. Error: %s", err)
		}
	default:
		response = "Неизвестная команда. Используй /help."
	}

	if !responseSent {
		message := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		if _, err := h.bot.Send(message); err != nil {
			slog.Error("error sending message", logger.Err(err))
		}
	}
}

func (h *Handler) handleAdd(ctx context.Context, fullName string) (string, error) {
	if strings.TrimSpace(fullName) == "" {
		return "Укажи репозиторий в формате owner/repo.\nПример: /add golang/go", nil
	}
	repo, err := h.service.Add(ctx, fullName)
	if err != nil {
		return "", fmt.Errorf("failed to add repository: %w", err)
	}
	return fmt.Sprintf("Репозиторий %s успешно добавлен.", repo.FullName), nil
}

func (h *Handler) handleDelete(ctx context.Context, fullName string) (string, error) {
	if strings.TrimSpace(fullName) == "" {
		return "Укажи удаляемый репозиторий в формате owner/repo.\nПример: /delete golang/go", errors.New("Укажи удаляемый репозиторий в формате owner/repo.\nПример: /delete golang/go")
	}

	if err := h.service.Delete(ctx, fullName); err != nil {
		return "ошибка удаления репозитория", fmt.Errorf("failed to delete repository: %w", err)
	}
	return fmt.Sprintf("репозиторий %s успешно удален", fullName), nil
}

func (h *Handler) handleList(ctx context.Context) (string, error) {
	repos, err := h.service.List(ctx)
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
	collectResult, err := h.service.Collect(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to collect repositories: %w", err)
	}

	switch {
	case collectResult.SuccessCount == 0 && collectResult.FailedCount == 0:
		return "Список репозиториев пуст. Добавь репозиторий через /add owner/repo.", nil
	case collectResult.SuccessCount == 0:
		return "Сбор статистики завершился с ошибкой для всех репозиториев.", nil
	case collectResult.FailedCount == 0:
		return fmt.Sprintf("Статистика всех репозиториев собрана успешно: %d.", collectResult.SuccessCount), nil
	default:
		return fmt.Sprintf("Сбор завершён. Успешно: %d, с ошибками: %d.", collectResult.SuccessCount,
			collectResult.FailedCount,
		), nil
	}
}

func (h *Handler) statsHandler(ctx context.Context) (string, error) {
	stats, err := h.service.Stats(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка получения статистики репозиториев: %w", err)
	}

	if len(stats) == 0 {
		return "Список отслеживаемых репозиториев пуст. Добавь первый: /add flecc1/URL-Shortener", nil
	}

	var response strings.Builder

	for _, stat := range stats {
		if stat.Err != nil {
			slog.Error("error getting repository snapshots", "repository", stat.Repository.FullName,
				logger.Err(stat.Err))
			fmt.Fprintf(&response, "Не удалось получить снимки репозитория %s.\n", stat.Repository.FullName)
			continue
		}

		if stat.Snapshot == nil {
			fmt.Fprintf(&response, "Репозиторий %s не имеет снимков.\n", stat.Repository.FullName)
			continue
		}

		snapshot := stat.Snapshot

		fmt.Fprintf(&response,
			"📦 %s\n"+
				"⭐ Stars: %d\n"+
				"🍴 Forks: %d\n"+
				"🐛 Open issues: %d\n"+
				"🔀 Open PRs: %d\n"+
				"💻 Commits за 7 дней: %d\n"+
				"📄 README: %t\n"+
				"🕒 Последний сбор: %s\n"+
				"🕒 Последний комит: %s\n"+
				"━━━━━━━━━━━━━━━━━━━━\n",
			stat.Repository.FullName,
			snapshot.Stars,
			snapshot.Forks,
			snapshot.OpenIssues,
			snapshot.OpenPullRequests,
			snapshot.CommitsLast7Days,
			snapshot.HasReadme,
			snapshot.CollectedAt,
			snapshot.LastCommitAt,
		)
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
		"/delete owner/repo - удалить репозиторий\n" +
		"/list — показать отслеживаемые репозитории\n" +
		"/collect — собрать и сохранить свежую статистику\n" +
		"/stats — показать последнюю сохранённую статистику"
}
