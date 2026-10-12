package bot

import (
	"github-watcher/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot               *tgbotapi.BotAPI
	repositoryService *repository.Service
}

func NewHandler(bot *tgbotapi.BotAPI, repositoryService *repository.Service) *Handler {
	return &Handler{
		bot:               bot,
		repositoryService: repositoryService,
	}
}
