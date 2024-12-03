package main

import (
	"ghost-approve/internal/app"
	"ghost-approve/internal/logging"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	log "github.com/sirupsen/logrus"
	_ "net/http/pprof"
)

func main() {
	logging.Init()
	log.Info("Starting bot...")

	if err := vkbot.InitBot(); err != nil {
		log.Fatalf("Falied to VK bot init: %v", err)
	}
	if err := postgres.InitDB(); err != nil {
		log.Fatalf("Falied to postgres init: %v", err)
	}

	if err := utils.InitLoader(); err != nil {
		log.Fatalf("Falied to loader init: %v", err)
	}

	if err := app.NewApp(vkbot.GetBot(), postgres.GetDB()).Run(); err != nil {
		log.Fatal(err)
	}
}
