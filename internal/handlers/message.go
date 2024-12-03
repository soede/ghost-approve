package handlers

import (
	"ghost-approve/internal/handlers/callbacks"
	"ghost-approve/internal/repositories"
	"ghost-approve/internal/services"
	commands "ghost-approve/internal/services/commands"
	"ghost-approve/pkg/vkbot"
	botgolang "github.com/mail-ru-im/bot-golang"
	"log"
	"strconv"
	"strings"
)

// MessageHandler обрабатывает входящие сообщения NEW_MESSAGE
func MessageHandler(p *botgolang.EventPayload) {
	message := p.Text
	userID := p.From.ID

	message = strings.TrimSpace(message)
	message = strings.Trim(message, "\u00A0")

	if message == "/start" {
		commands.Start(p)
		return
	}

	//проверка регистрации
	user := repositories.GetUserByID(userID)
	if user == nil {
		repositories.CreateUser(userID, p.From.FirstName, p.From.LastName)
	}
	if user != nil && !user.Registered {
		err := repositories.ActivateUser(user, p.From.FirstName, p.From.LastName)
		if err != nil {
			log.Println(err)
		}
	}

	//проверка на наличие файла, если есть – подтвердить апрув
	if fileInfo, exist := services.WaitForFile[userID]; exist {
		callbacks.CheckAndConfirm(p, fileInfo)
		return
	}

	if strings.HasPrefix(message, "/received") {
		if strings.Contains(message, "_") {
			approveID, err := strconv.Atoi(strings.TrimPrefix(message, "/received_"))
			if err != nil {
				vkbot.GetBot().NewTextMessage(userID, "Я не смог понять ID апрува :(")
				return
			}
			commands.Received(userID, approveID)
			return
		}
		commands.Received(userID, 0)
		return
	}

	if strings.HasPrefix(message, "/report") {
		approveID, err := strconv.Atoi(strings.TrimPrefix(message, "/report"))
		if err != nil {
			return
		}
		if access, err := services.CheckApprovalAccess(approveID, userID); !access || err != nil {
			vkbot.GetBot().NewTextMessage(userID, "У тебя нет доступа к этому апруву").Send()
			return
		}

		message, err := services.ReportMessage(approveID, userID)
		if err != nil {
			log.Println(err)
		}
		err = message.Send()
		if err != nil {
			log.Println(err)
		}
		return

	}

	switch message {
	case "/cancel":
		commands.Cancel(p)
		return
	case "/create":
		commands.Create(p)
		return
	case "/check":
		commands.Check(p)
		return
	case "/manage":
		commands.Manage(p)
		return

	default:
		if commands.UserStates[userID] != nil {
			commands.Create(p)
		}

		if commands.UserStates[userID] == nil {
			err := vkbot.GetBot().NewTextMessage(userID, "Я не понял что ты имел ввиду\nНо если отправишь мне /start, то я расскажу тебе про свои основные команды").Send()
			if err != nil {
				log.Println(err)
			}
		}

	}
}
