package notifier

import (
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	"log"
	"time"
)

func CheckTasks(taskCheckChan chan<- *models.Task) {
	for {
		time.Sleep(1 * time.Minute)

		tasks, err := repositories.CurrentTasks()
		if err != nil {
			log.Printf("Error checking Redis tasks: %v", err)
			continue
		}

		for _, task := range tasks {
			taskCheckChan <- &task
		}
	}
}
