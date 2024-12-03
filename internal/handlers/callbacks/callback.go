package callbacks

import (
	"fmt"
	"ghost-approve/internal/services"
	"ghost-approve/internal/services/commands"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
	"strings"
)

// CallbackHandler обрабатывает все входящие CALLBACK_QUERY события
func CallbackHandler(p *botgolang.EventPayload) {
	userID := p.From.ID
	data := p.CallbackQuery().CallbackData

	if isConfirmCommand(data) {
		handleConfirmCallback(p, data)
		return
	}

	if isRejectCommand(data) {
		handleRejectCallback(p, data)
		return
	}

	if isChangeCommand(userID, data) {
		handleChangeCallback(p)
	}

	if isManageCommand(data) {
		command := strings.TrimPrefix(data, "/manage_")
		handleManageCommand(command, p.From.ID)
	}

	if isCreateCommand(userID, data) {
		CreateCallback(p)
	}

	//отменяет подтверждение файла
	if data == "/waitFile_cancel" {
		var err error
		if el, exist := services.WaitForFile[userID]; exist {
			err = vkbot.GetBot().NewTextMessage(userID, fmt.Sprintf("Подтверждение апрува #%d было отменено", el.ApproveID)).Send()
			delete(services.WaitForFile, userID)
		} else {
			err = vkbot.GetBot().NewTextMessage(userID, "Ты не начал подтверждать апрув").Send()
		}
		if err != nil {
			log.Println(err)
		}
	}
}

func isConfirmCommand(data string) bool {
	return strings.HasPrefix(data, "/confirm_")
}

func isRejectCommand(data string) bool {
	return strings.HasPrefix(data, "/reject_")
}

func isChangeCommand(userId string, data string) bool {
	if _, exists := commands.UserStates[userId]; !exists {
		return false
	}

	if commands.UserStates[userId].CurrentStage != commands.ExampleMessage {
		return false
	}
	return strings.HasPrefix(data, "/change_") || data == "/not_change"
}

func isManageCommand(data string) bool {
	return strings.HasPrefix(data, "/manage_")
}

func isCreateCommand(userId, data string) bool {
	if _, exists := commands.UserStates[userId]; !exists {
		return false
	}
	return strings.HasPrefix(data, "/create_")
}
