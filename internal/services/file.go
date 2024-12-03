package services

import (
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

func UploadFileWithLock(approveID, version int, uploaderID, newFileID string) error {
	return postgres.GetDB().Transaction(func(tx *gorm.DB) error {
		var existingFile models.File
		if err := tx.Raw(`
			SELECT * 
			FROM files 
			WHERE approve_id = ? 
			FOR UPDATE
		`, approveID).Scan(&existingFile).Error; err != nil {
			return fmt.Errorf("failed to lock row: %w", err)
		}
		if version == existingFile.Version {
			return botErrors.FileLockedError
		}

		existingFile.UploaderID = uploaderID
		existingFile.FileID = newFileID
		existingFile.Version = version
		existingFile.UploadedAt = time.Now()

		if err := tx.Save(&existingFile).Error; err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}

		err := ConfirmApprove(approveID, 0, uploaderID)
		delete(WaitForFile, uploaderID)

		if err != nil {
			return err
		}

		history := models.FileHistory{
			FileID:     existingFile.FileID,
			ApproveID:  approveID,
			UploaderID: uploaderID,
			Version:    existingFile.Version,
			UploadedAt: existingFile.UploadedAt,
		}
		if err := tx.Create(&history).Error; err != nil {
			return fmt.Errorf("failed to save history: %w", err)
		}

		return nil
	})
}

func SendFileOrCancel(userID, text string) {
	message := vkbot.GetBot().NewTextMessage(userID, text+"\nЕсли ты передумал отправлять мне файл, нажми кнопку Отменить")
	not := botgolang.NewCallbackButton("Отменить", "/waitFile_cancel").WithStyle(botgolang.ButtonAttention)
	keyboard := botgolang.NewKeyboard()
	keyboard.AddRow(not)
	message.AttachInlineKeyboard(keyboard)
	err := message.Send()
	if err != nil {
		log.Error(err)
	}
}
