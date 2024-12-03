package main

import (
	"ghost-approve/internal/app"
	"ghost-approve/internal/logs"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	"log"
	_ "net/http/pprof"
)

func main() {
	logs.SetupLogging()
	log.Println("Starting bot...")
	
	if err := vkbot.InitBot(); err != nil {
		log.Fatalf("Falied to VK bot init: %v", err)
	}
	if err := postgres.InitDB(); err != nil {
		log.Fatalf("Falied to postgres init: %v", err)
	}
	/*	removeAll := true
		if removeAll {
			postgres.Drop()
		}*/

	if err := utils.InitLoader(); err != nil {
		log.Fatalf("Falied to loader init: %v", err)
	}

	if err := app.NewApp(vkbot.GetBot(), postgres.GetDB()).Run(); err != nil {
		log.Fatal(err)
	}
}
