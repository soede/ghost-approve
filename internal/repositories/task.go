package repositories

import (
	"fmt"
	"ghost-approve/internal/models"
	rdb "ghost-approve/pkg/db/redis"
	"log"
	"strconv"
	"strings"
	"time"
)

// CreateTasks – создает напоминания и завершение апрува в redis
func CreateTasks(approvalID int, confirmTime int) error {
	var err error
	rdbClient := rdb.Client()

	now := time.Now()
	halfTime := now.Add(time.Duration(confirmTime*60/2) * time.Minute).Unix()
	quarterTime := now.Add(time.Duration(confirmTime*60-confirmTime*60/4) * time.Minute).Unix()
	endTime := now.Add(time.Duration(confirmTime) * time.Hour).Unix()

	err = rdbClient.ZAdd(
		"approve_notifications",
		fmt.Sprintf("half:%d", approvalID),
		float64(halfTime))
	err = rdbClient.ZAdd("approve_notifications",
		fmt.Sprintf("quarter:%d", approvalID),
		float64(quarterTime))
	err = rdbClient.ZAdd("approve_notifications",
		fmt.Sprintf("end:%d", approvalID),
		float64(endTime))

	return err

}
func CurrentTasks() ([]models.Task, error) {
	tasks, err := rdb.Client().ZRangeByScoreWithScores("approve_notifications", "0", fmt.Sprintf("%d", time.Now().Unix()))
	if err != nil {
		return nil, err
	}

	var taskList []models.Task
	for _, task := range tasks {
		parts := strings.Split(task.Member.(string), ":")
		if len(parts) != 2 {
			continue
		}

		approveID, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("Ошибка преобразования ID задачи: %v", err)
			continue
		}
		taskList = append(taskList, models.Task{
			ApproveID: approveID,
			Member:    task.Member.(string),
		})
	}

	return taskList, nil
}

func NextApprovalTask(approvalID int) float64 {
	halfKey := fmt.Sprintf("half:%d", approvalID)
	quarterKey := fmt.Sprintf("quarter:%d", approvalID)

	halfResult, err := rdb.Client().ZScore("approve_notifications", halfKey)

	if err == nil {
		return halfResult
	}

	quarterResult, err := rdb.Client().ZScore("approve_notifications", quarterKey)

	if err == nil {
		return quarterResult
	}

	return -1

}
