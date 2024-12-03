package commands

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/utils"
	"ghost-approve/pkg/db/postgres"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"gorm.io/gorm"
	"log"
	"strings"
	"sync"
	"time"
)

// Показывает все отчеты, которые были получены пользователем
func Received(userID string, approvalID int) {
	if approvalID != 0 {
		approval, err := repositories.ReceivedApprovalByID(userID, approvalID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err := vkbot.GetBot().NewTextMessage(userID, "Я не смог получить этот апрув. Возможно, он не был создан").Send()
			if err != nil {
				return
			}
		}
		if err != nil {
			log.Println(err)
		}

		if approval == nil {
			vkbot.GetBot().NewTextMessage(userID, "Я не смог найти этот апрув :(")
		}

		/*if approval.IsPrivate {
			answer := generatePrivateMessage(userID, approvalID)
			err := sendAll([]*botgolang.Message{answer})
			if err != nil {
				log.Println(err)
			}
			return
		}*/

		if err != nil {
			log.Println(err)
		}
		return

	}

	approvals, err := repositories.ReceivedApprovals(userID)
	if err != nil {
		log.Println(err)
		return
	}

	if len(approvals) == 0 {
		err := vkbot.GetBot().NewTextMessage(userID, "У тебя пока что нет полученных апрувов").Send()
		if err != nil {
			return
		}
		return
	}

	var wg sync.WaitGroup
	messageChan := make(chan *botgolang.Message, len(approvals))

	for _, approval := range approvals {

		if approval.AuthorID == userID {
			continue
		}

		wg.Add(1)

		go func(el models.Approval) {
			defer wg.Done()
			//проверка, не скрыл ли пользователь апрув
			if hidden, err := repositories.IsReportHiddenForUser(userID, el.ID); hidden && err == nil {
				return
			}

			//проверка, не приватный ли апрув
			if el.IsPrivate {
				message := generatePrivateMessage(&el, userID)
				if el.Editable {

				}
				keyboard := botgolang.NewKeyboard()
				hide := botgolang.NewCallbackButton("Скрыть", fmt.Sprintf("/manage_hide_%d", el.ID))
				keyboard.AddRow(hide)
				message.AttachInlineKeyboard(keyboard)

				messageChan <- message
				return
			}

			if err != nil {
				log.Println(err)
			}

			message := receivedApproveMessage(&el, userID)
			keyboard := botgolang.NewKeyboard()
			statistic := botgolang.NewCallbackButton("📊Статистика", fmt.Sprintf("/manage_statistic_%d", el.ID))
			events := botgolang.NewCallbackButton("🕒 История действий", fmt.Sprintf("/manage_events_%d", el.ID))
			hide := botgolang.NewCallbackButton("Скрыть", fmt.Sprintf("/manage_hide_%d", el.ID))
			keyboard.AddRow(statistic, events)
			keyboard.AddRow(hide)

			message.AttachInlineKeyboard(keyboard)
			messageChan <- message

		}(*approval)

	}
	wg.Wait()

	go func() {
		wg.Wait()
		close(messageChan)
	}()

	var sendWg sync.WaitGroup
	for message := range messageChan {
		sendWg.Add(1)
		go func(msg *botgolang.Message) {
			defer sendWg.Done()
			err := msg.Send()
			if err != nil {
				log.Println("Ошибка при отправке сообщения:", err)
			}
		}(message)
	}
	sendWg.Wait()

}

// генерирует сообщение по приватному апруву
func generatePrivateMessage(approval *models.Approval, userID string) *botgolang.Message {
	message := vkbot.GetBot().NewMessage(userID)
	text := fmt.Sprintf("Апрув #%d", approval.ID) + "\n"
	text += approval.Title + "\n"
	if len(approval.Description) != 0 {
		text += "Описание: " + approval.Description + "\n"
	}

	text += "Автор: " + utils.CreateUserLink([]string{approval.AuthorID}) + "\n"
	text += "Было отправлено: " + utils.FormatCreatedAt(&approval.CreatedAt) + "\n"
	response, err := userReaction(approval.ID, userID, approval.Editable)
	if err == nil {
		text += response + "\n"
	}
	text += "\n"
	text += "\n🔒Автор апрува решил сделать его приватным, у тебя нет доступа к статистике"
	var file *models.File
	result := postgres.GetDB().Where("approve_id = ?", approval.ID).First(&file)
	if result != nil && result.Error == nil {
		message.FileID = file.OriginalFileID
	}
	message.Text = text

	return message
}

func sendAll(messages []*botgolang.Message) (err error) {
	for _, message := range messages {
		err = message.Send()
	}
	return
}

// сообщение для публичного апрува
func receivedApproveMessage(approve *models.Approval, userID string) *botgolang.Message {
	exampleMessage := vkbot.GetBot().NewMessage(userID)
	var file *models.File
	result := postgres.GetDB().Where("approve_id = ?", approve.ID).First(&file)
	if result != nil && result.Error == nil {
		exampleMessage.FileID = file.FileID
	}

	exampleMessage.Text = receivedApproveText(approve, userID)
	keyboard := botgolang.NewKeyboard()

	exampleMessage.AttachInlineKeyboard(keyboard)
	return exampleMessage
}

// собирует текст для сообщения в /received
func receivedApproveText(approval *models.Approval, userID string) string {
	var exampleText string
	exampleText += fmt.Sprintf("Апрув #%d", approval.ID)
	exampleText += fmt.Sprintf("\n%s", approval.Title)

	if len(approval.Description) != 0 {
		exampleText += "\nОписание: " + approval.Description
	}
	authorLink := utils.CreateUserLink([]string{approval.AuthorID})
	exampleText += fmt.Sprintf("\nОт: %s", authorLink)

	exampleText += fmt.Sprintf("\nБыло отправлено: %s", utils.FormatCreatedAt(&approval.CreatedAt))

	targetTime := approval.CreatedAt.Add(time.Duration(approval.ConfirmTime) * time.Hour)
	currentTime := time.Now()
	timeRemaining := int(targetTime.Sub(currentTime).Minutes())

	if timeRemaining <= 0 || approval.Status != models.StatusPending {
		exampleText += fmt.Sprintf("\n\nСтатус: %s", approval.Status)
	}
	if timeRemaining > 0 && approval.Status == models.StatusPending {
		exampleText += "\nОсталось времени: " + utils.FormatMinutes(timeRemaining)
	}

	if len(approval.Links) != 0 {
		exampleText += "\nПрикрепленные ссылки: "

		exampleText += strings.Join(approval.Links, ", ")
	}

	if approval.Cancelable {
		exampleText += "\nТип апрува: отклоняемый"
		if approval.StopOnReject {
			exampleText += ", будет завершен после первого отклонения"
		}
		if !approval.StopOnReject {
			exampleText += ", не будет завершен после первого отклонения"
		}
	}
	if !approval.Cancelable {
		exampleText += "\nТип апрува: не отклоняемый"

	}

	if approval.Editable {
		exampleText += "\nТип файла: редактируемый, каждый участник должен отправить его обновленную версию"
	}

	response, err := userReaction(approval.ID, userID, approval.Editable)
	if err == nil {
		exampleText += fmt.Sprintf("\n\n%s", response)
	}

	return exampleText

}

// находит реакцию пользователя на апрув
func userReaction(approvalID int, userID string, isEditable bool) (string, error) {
	var err error

	rejected, err := repositories.IsUserRejected(approvalID, userID)
	approved, err := repositories.IsUserApproved(approvalID, userID)

	if rejected {
		return "❌Ты отклонил этот апрув", nil
	}

	if approved {
		if isEditable {
			file, err := repositories.FileByUploader(userID, approvalID)
			if file != nil && err == nil {
				url, err := utils.FileUrlByID(file.FileID)
				if url != "" && err == nil {
					return fmt.Sprintf("📝Ты подтвердил этот апрув и прикрепил файл: %s", url), nil
				}
			}
		}
		return "✅Ты подтвердил этот апрув", nil
	}

	if err != nil {
		return "", err
	}
	return "Ты не дал отклика по этому апруву", nil

}
