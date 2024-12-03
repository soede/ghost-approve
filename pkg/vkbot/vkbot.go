package vkbot

import (
	"github.com/joho/godotenv"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"os"
)

var bot *botgolang.Bot

func InitBot() error {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	token := os.Getenv("VK_BOT_TOKEN")

	bot, err = botgolang.NewBot(token, botgolang.BotDebug(true))
	if err != nil {
		log.Fatalf("Falied connect to bot: %s", err)
	}
	return err
}

func GetBot() *botgolang.Bot {
	return bot
}
