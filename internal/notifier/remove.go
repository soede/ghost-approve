package notifier

import (
	"ghost-approve/internal/models"

	"ghost-approve/pkg/db/redis"
	"log"
)

// RemoveTask – удаляет истекшие апрувы и уведомляет пользователей
func RemoveTask(taskResult *models.Task) {
	err := SendRemindMessage(taskResult)
	log.Printf("Received task check result: %s", taskResult)
	if taskResult == nil {
		return
	}
	err = redis.Client().ZRem("approve_notifications", taskResult.Member)
	if err != nil {
		log.Println(err)
	}
}
