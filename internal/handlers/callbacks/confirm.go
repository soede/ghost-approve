package callbacks

import (
	"errors"
	"fmt"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/services"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"gorm.io/gorm"
	"log"
)

// handleConfirmCallback обрабатывает confirm_* команды
func handleConfirmCallback(p *botgolang.EventPayload, data string) {
	approveID, v, err := utils.ParseFileInfo(data)
	if err != nil {
		log.Println(err)
		return
	}

	if err != nil {
		log.Printf("Ошибка преобразования ID: %s в int: %v", approveID, err)
	}

	err = services.ConfirmApprove(approveID, v, p.From.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Апрув не был найден").Send()
		log.Println(err)
	}
	if errors.Is(err, botErrors.ErrAlreadyHasResponse) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Ты уже дал отклик по этому апруву").Send()
	}

	if errors.Is(err, botErrors.ErrNoAccess) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Нет доступа к апруву").Send()
		log.Println(err)
	}

	if err != nil {
		log.Printf("Ошибка подтверждения: %v", err)
	}

}

func CheckAndConfirm(p *botgolang.EventPayload, fileInfo *services.FileInfo) {
	var fileID string
	var ok bool
	for _, pts := range p.Parts {
		if pts.Type == botgolang.FILE {
			fileID = pts.Payload.FileID
			ok = true
		}
	}
	if !ok {
		message := vkbot.GetBot().NewTextMessage(p.From.ID, "Ты отправил сообшение в котором нет файла, давай попробуем заново")
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return
	}

	//не трогать, очень важно
	file, err := repositories.FileByApprovalID(fileInfo.ApproveID)

	if err != nil {
		log.Println(err)
	}

	err = services.UploadFileWithLock(fileInfo.ApproveID, fileInfo.Version+1, p.From.ID, fileID)

	if errors.Is(err, botErrors.FileLockedError) {
		file, err = repositories.FileByApprovalID(fileInfo.ApproveID)
		if err != nil {
			log.Println("ошибка при загрузке")
		}
		message := vkbot.GetBot().NewMessage(p.From.ID)
		message.Text = "Ты отправил не актуальную версию файла. Попробуй  скачать последнюю версию файла, изменить по своему усмотрению и отправить мне его. \n" +
			"Прикрепил последнюю версию файла 👻\n" +
			"Если передумал отправлять мне файл, нажми кнопку 'Отменить'"
		message.FileID = file.FileID

		fileInfo.Version = file.Version

		not := botgolang.NewCallbackButton("Отменить", "/waitFile_cancel").WithStyle(botgolang.ButtonAttention)
		keyboard := botgolang.NewKeyboard()
		keyboard.AddRow(not)
		message.AttachInlineKeyboard(keyboard)
		if err := message.Send(); err != nil {
			log.Printf("failed to send message: %s", err)
		}
		return
	}
	if errors.Is(err, botErrors.ErrNoAccess) {
		vkbot.GetBot().NewTextMessage(p.From.ID, "Нет доступа к апруву").Send()
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		vkbot.GetBot().NewTextMessage(p.From.ID, "Апрув не был найден").Send()
		return
	}
	if errors.Is(err, botErrors.ErrAlreadyHasResponse) {
		err = vkbot.GetBot().NewTextMessage(p.From.ID, "Ты уже дал отклик по этому апруву").Send()
		return
	}
	if err != nil {
		log.Println(err)
	}

	err = vkbot.GetBot().NewTextMessage(p.From.ID, fmt.Sprintf("Апрув #%d был успешно подтвержден!", fileInfo.ApproveID)).Send()

	return
}
