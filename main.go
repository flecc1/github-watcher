package main

import (
	"log"
	"log/slog"
	"os"

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
