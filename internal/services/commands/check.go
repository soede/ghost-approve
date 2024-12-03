package commands

import (
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	utlis "ghost-approve/internal/utils"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
	"time"
)

func Check(a *botgolang.EventPayload) {
	var pendingApprovals []models.Approval
	var err error
	var userID = a.From.ID

	pendingApprovals, err = repositories.FindPendingApprovalsByUserID(userID)
	if err != nil {
		log.Println(err)
	}

	if len(pendingApprovals) == 0 {
		err := vkbot.GetBot().NewTextMessage(userID, "Пока что у тебя нет актуальных апрувов").Send()
		if err != nil {
			log.Println("Не удалось отравить")
		}
		return
	}

	for _, el := range pendingApprovals {
		go func(approve models.Approval) {
			if err := ApproveMessage(&approve, userID).Send(); err != nil {
				log.Println("Ошибка отправки сообщения:", err)
			}
		}(el)
	}

}

// ApproveMessage – составляет сообщение для апрува
func ApproveMessage(approve *models.Approval, userID string) *botgolang.Message {
	exampleMessage := vkbot.GetBot().NewMessage(userID)
	var file *models.File
	result := postgres.GetDB().Where("approve_id = ?", approve.ID).First(&file)
	if result != nil && result.Error == nil {
		exampleMessage.FileID = file.FileID
	}

	exampleMessage.Text = GenerateApproveText(approve)
	keyboard := botgolang.NewKeyboard()
	if len(approve.Links) != 0 {
		var links []botgolang.Button
		for id, el := range approve.Links {
			link := botgolang.NewURLButton(fmt.Sprintf("#%v %s", id, "Ссылка"), el)
			links = append(links, link)
		}
		keyboard.AddRow(links...)
	}

	confirmCallback := fmt.Sprintf("/confirm_%d", approve.ID)

	if approve.Editable && file.FileID != "" {
		confirmCallback = fmt.Sprintf("/confirm_%d_%d", approve.ID, file.Version)

	}

	yesButton := botgolang.NewCallbackButton("✅Подтвердить", confirmCallback)
	if approve.Cancelable {
		noButton := botgolang.NewCallbackButton("⛔️Отклонить", fmt.Sprintf("/reject_%d", approve.ID))
		keyboard.AddRow(yesButton, noButton)
	} else {
		keyboard.AddRow(yesButton)
	}
	exampleMessage.AttachInlineKeyboard(keyboard)
	return exampleMessage
}

func GenerateApproveText(approve *models.Approval) string {
	var exampleText string
	exampleText += fmt.Sprintf("Апрув #%d", approve.ID)
	exampleText += fmt.Sprintf("\n%s", approve.Title)

	if len(approve.Description) != 0 {
		exampleText += "\nОписание: " + approve.Description
	}
	authorLink := utlis.CreateUserLink([]string{approve.AuthorID})
	exampleText += fmt.Sprintf("\nОт: %s", authorLink)

	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}
	ca := approve.CreatedAt
	exampleText += fmt.Sprintf("\nБыло отправлено: %02d %s %02d:%02d",
		ca.Day(), months[ca.Month()-1], ca.Hour(), ca.Minute())

	targetTime := approve.CreatedAt.Add(time.Duration(approve.ConfirmTime) * time.Hour)
	currentTime := time.Now()
	timeRemaining := int(targetTime.Sub(currentTime).Minutes())
	exampleText += "\nОсталось времени: " + utlis.FormatMinutes(timeRemaining)

	if approve.Editable {
		exampleText += "\n\nОтправь отредактированную версию документа для подтверждения. Когда будешь готов, нажми \"подтвердить\" "
	}
	return exampleText

}
