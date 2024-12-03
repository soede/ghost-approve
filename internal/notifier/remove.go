package notifier

import (
	"ghost-approve/internal/models"
	"ghost-approve/pkg/db/redis"
	log "github.com/sirupsen/logrus"
)

// RemoveTask – удаляет истекшие апрувы и уведомляет пользователей
func RemoveTask(taskResult *models.Task) {
	err := SendRemindMessage(taskResult)
	log.Info("Received task check result: %s", taskResult)
	if taskResult == nil {
		return
	}
	err = redis.Client().ZRem("approve_notifications", taskResult.Member)
	if err != nil {
		log.Errorf("error when deleting task %s: %s", taskResult.ApproveID, err)
	}
}
