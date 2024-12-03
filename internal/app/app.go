package app

import (
	"context"
	"ghost-approve/internal/models"
	"ghost-approve/internal/notifier"
	"ghost-approve/pkg/db/redis"
	botgolang "github.com/mail-ru-im/bot-golang"
	"gorm.io/gorm"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type App struct {
	psql *gorm.DB
	bot  *botgolang.Bot
}

func NewApp(bot *botgolang.Bot, psql *gorm.DB) *App {
	return &App{psql, bot}
}

func (a *App) Run() error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redis.InitRedis(ctx)
	notifier.InitNotifier(a.bot)

	taskCheckChan := make(chan *models.Task)
	go notifier.CheckTasks(taskCheckChan)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	updates := a.bot.GetUpdatesChannel(ctx)

	for {
		select {
		case update := <-updates:
			Updates(&update.Type, &update.Payload)
		case taskResult := <-taskCheckChan:
			notifier.RemoveTask(taskResult)
		case sig := <-quit:
			log.Printf("Received signal: %s. Shutting down...", sig)
			cancel()
			return nil
		}
	}

}

