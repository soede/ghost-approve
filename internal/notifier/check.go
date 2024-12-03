package notifier

import (
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	log "github.com/sirupsen/logrus"
	"time"
)

func CheckTasks(taskCheckChan chan<- *models.Task) {
	for {
		time.Sleep(1 * time.Minute)

		tasks, err := repositories.CurrentTasks()
		if err != nil {
			log.Errorf("Error checking Redis tasks: %v", err)
			continue
		}

		for _, task := range tasks {
			taskCheckChan <- &task
		}
	}
}
