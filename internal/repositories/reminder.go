package repositories

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/db/postgres"
	"gorm.io/gorm"
	"time"
)

func CreateRemind(approvalID int) error {
	newReminder := models.ApprovalReminder{
		ApprovalID: approvalID,
		CreatedAt:  time.Now(),
	}

	return postgres.GetDB().Create(&newReminder).Error
}

func IsReminderExpired(approvalID int) (bool, error) {
	var lastReminder models.ApprovalReminder
	err := postgres.GetDB().Where("approval_id = ?", approvalID).Order("created_at desc").First(&lastReminder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}

	timeDiff := time.Since(lastReminder.CreatedAt)
	if timeDiff >= time.Hour {
		return true, nil
	}

	return false, nil
}

// CountApprovalReminders – считает количество отправленных напоминаний
func CountApprovalReminders(approveID int) (int64, error) {
	var count int64
	err := postgres.GetDB().Model(&models.ApprovalReminder{}).Where("approval_id = ?", approveID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func ApprovalRemindersByID(approvalID int) (*[]models.ApprovalReminder, error) {
	var reminders []models.ApprovalReminder

	err := postgres.GetDB().Raw(`
		SELECT ar.* 
		FROM approval_reminders ar
		WHERE ar.approval_id = ?
	`, approvalID).Scan(&reminders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch approval reminders: %w", err)
	}

	return &reminders, nil
}
