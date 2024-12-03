package repositories

import (
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/db/postgres"
)

func FileByID(id int) (*models.File, error) {
	var file *models.File
	err := postgres.GetDB().Raw(`SELECT * FROM files WHERE id = ?`, id).Scan(&file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func FileByApprovalID(approveID int) (*models.File, error) {
	var file *models.File
	err := postgres.GetDB().Raw(`SELECT * FROM files WHERE approve_id = ? LIMIT 1`, approveID).Scan(&file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

func GetFileHistoriesByID(approveID int) (*[]models.FileHistory, error) {
	var fileHistories []models.FileHistory

	err := postgres.GetDB().Raw(`
		SELECT fh.* 
		FROM file_histories fh
		WHERE fh.approve_id = ?
	`, approveID).Scan(&fileHistories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch file histories: %w", err)
	}

	return &fileHistories, nil
}

func FileByUploader(uploaderID string, approveID int) (*models.FileHistory, error) {
	var fileHistory models.FileHistory
	err := postgres.GetDB().Where("uploader_id = ? AND approve_id = ?", uploaderID, approveID).First(&fileHistory).Error
	if err != nil {
		return nil, err
	}
	return &fileHistory, nil
}
