package repositories

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/db/postgres"
	"gorm.io/gorm"
)

func ReceivedApprovalByID(userID string, approvalID int) (*models.Approval, error) {
	var approval *models.Approval

	query := `
		SELECT a.*
		FROM approvals a
		JOIN approval_users au ON a.id = au.approval_id
		WHERE a.id = ? AND au.user_id = ?
	`

	err := db.Raw(query, approvalID, userID).Scan(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	// Если апрув найден, возвращаем его
	return approval, nil
	return nil, nil
}

// возвращает полученные пользователем апрувы
func ReceivedApprovals(userID string) ([]*models.Approval, error) {
	var approvals []*models.Approval

	query := `
		SELECT a.*
		FROM approvals a
		JOIN approval_users au ON a.id = au.approval_id
		WHERE au.user_id = ?
	`

	err := postgres.GetDB().Raw(query, userID).Scan(&approvals).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching approvals: %v", err)
	}

	return approvals, nil
}

// Прячет репорт для пользователя
func HideReportForUser(userID string, approveID int) error {
	hiddenReport := models.HiddenReport{
		UserID:    userID,
		ApproveID: approveID,
	}

	if err := postgres.GetDB().Where("user_id = ? AND approve_id = ?", userID, approveID).First(&hiddenReport).Error; err == nil {
		return nil
	}

	return postgres.GetDB().Create(&hiddenReport).Error
}

// Проверяет, не скрыл ли пользователь этот апрув
func IsReportHiddenForUser(userID string, approveID int) (bool, error) {
	var hiddenReport models.HiddenReport
	err := postgres.GetDB().Where("user_id = ? AND approve_id = ?", userID, approveID).First(&hiddenReport).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil // Отчет не скрыт
	}
	return err == nil, err
}
